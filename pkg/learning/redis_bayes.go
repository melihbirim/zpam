package learning

import (
	"context"
	"crypto/sha1"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBayesianFilter implements Rspamd-style Bayesian filtering with Redis backend
type RedisBayesianFilter struct {
	client    *redis.Client
	config    *RedisConfig
	ctx       context.Context
	tokenizer *OSBTokenizer
}

// RedisConfig holds Redis Bayesian configuration
type RedisConfig struct {
	// Redis connection
	RedisURL    string `json:"redis_url" yaml:"redis_url"`
	KeyPrefix   string `json:"key_prefix" yaml:"key_prefix"`
	DatabaseNum int    `json:"database_num" yaml:"database_num"`

	// Tokenization (Rspamd-style OSB)
	OSBWindowSize  int `json:"osb_window_size" yaml:"osb_window_size"`
	MinTokenLength int `json:"min_token_length" yaml:"min_token_length"`
	MaxTokenLength int `json:"max_token_length" yaml:"max_token_length"`
	MaxTokens      int `json:"max_tokens" yaml:"max_tokens"`

	// Learning parameters
	MinLearns     int     `json:"min_learns" yaml:"min_learns"`
	MaxLearns     int     `json:"max_learns" yaml:"max_learns"`
	SpamThreshold float64 `json:"spam_threshold" yaml:"spam_threshold"`

	// Per-user support
	PerUserStats bool   `json:"per_user_stats" yaml:"per_user_stats"`
	DefaultUser  string `json:"default_user" yaml:"default_user"`

	// Token expiration (like Rspamd)
	TokenTTL        time.Duration `json:"token_ttl" yaml:"token_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// Performance
	LocalCache bool          `json:"local_cache" yaml:"local_cache"`
	CacheTTL   time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	BatchSize  int           `json:"batch_size" yaml:"batch_size"`
}

// DefaultRedisConfig returns default Redis configuration
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		RedisURL:        "redis://localhost:6379",
		KeyPrefix:       "zpo:bayes",
		DatabaseNum:     0,
		OSBWindowSize:   5,
		MinTokenLength:  3,
		MaxTokenLength:  32,
		MaxTokens:       1000,
		MinLearns:       200,
		MaxLearns:       5000,
		SpamThreshold:   0.95,
		PerUserStats:    true,
		DefaultUser:     "global",
		TokenTTL:        30 * 24 * time.Hour, // 30 days
		CleanupInterval: 6 * time.Hour,
		LocalCache:      true,
		CacheTTL:        5 * time.Minute,
		BatchSize:       100,
	}
}

// OSBTokenizer implements Orthogonal Sparse Bigrams like Rspamd
type OSBTokenizer struct {
	config *RedisConfig
}

// NewRedisBayesianFilter creates a new Redis-backed Bayesian filter
func NewRedisBayesianFilter(config *RedisConfig) (*RedisBayesianFilter, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %v", err)
	}

	opt.DB = config.DatabaseNum
	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis connection failed: %v", err)
	}

	rbf := &RedisBayesianFilter{
		client:    client,
		config:    config,
		ctx:       ctx,
		tokenizer: &OSBTokenizer{config: config},
	}

	// Start cleanup routine if enabled
	if config.CleanupInterval > 0 {
		go rbf.startCleanupRoutine()
	}

	return rbf, nil
}

// OSB Token generation (like Rspamd)
func (ot *OSBTokenizer) GenerateOSBTokens(text string) []string {
	// Normalize text
	text = strings.ToLower(text)
	text = regexp.MustCompile(`[^\p{L}\p{N}\s]+`).ReplaceAllString(text, " ")

	// Split into words
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(text), -1)

	var tokens []string

	// Generate unigrams
	for _, word := range words {
		if len(word) >= ot.config.MinTokenLength && len(word) <= ot.config.MaxTokenLength {
			tokens = append(tokens, word)
		}
	}

	// Generate OSB bigrams
	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words) && j <= i+ot.config.OSBWindowSize; j++ {
			word1, word2 := words[i], words[j]
			if len(word1) >= ot.config.MinTokenLength && len(word2) >= ot.config.MinTokenLength {
				// Create OSB token with position information
				osb := fmt.Sprintf("%s|%s|%d", word1, word2, j-i)
				tokens = append(tokens, osb)
			}
		}
	}

	// Limit tokens to prevent abuse
	if len(tokens) > ot.config.MaxTokens {
		tokens = tokens[:ot.config.MaxTokens]
	}

	return tokens
}

// TrainSpam trains the filter on spam content
func (rbf *RedisBayesianFilter) TrainSpam(subject, body, user string) error {
	return rbf.train(subject, body, user, true)
}

// TrainHam trains the filter on ham content
func (rbf *RedisBayesianFilter) TrainHam(subject, body, user string) error {
	return rbf.train(subject, body, user, false)
}

// train internal training method
func (rbf *RedisBayesianFilter) train(subject, body, user string, isSpam bool) error {
	if user == "" {
		user = rbf.config.DefaultUser
	}

	// Generate tokens
	text := subject + " " + body
	tokens := rbf.tokenizer.GenerateOSBTokens(text)

	if len(tokens) == 0 {
		return nil
	}

	// Create Redis pipeline for batch operations
	pipe := rbf.client.Pipeline()

	userKey := rbf.getUserKey(user)
	tokenType := "ham"
	if isSpam {
		tokenType = "spam"
	}

	// Update token counts
	for _, token := range tokens {
		tokenKey := rbf.getTokenKey(user, token)

		// Increment token count
		pipe.HIncrBy(rbf.ctx, tokenKey, tokenType, 1)

		// Set expiration
		if rbf.config.TokenTTL > 0 {
			pipe.Expire(rbf.ctx, tokenKey, rbf.config.TokenTTL)
		}
	}

	// Update global counters
	if isSpam {
		pipe.HIncrBy(rbf.ctx, userKey, "spam_learned", 1)
		pipe.HIncrBy(rbf.ctx, userKey, "spam_tokens", int64(len(tokens)))
	} else {
		pipe.HIncrBy(rbf.ctx, userKey, "ham_learned", 1)
		pipe.HIncrBy(rbf.ctx, userKey, "ham_tokens", int64(len(tokens)))
	}

	// Update last trained timestamp
	pipe.HSet(rbf.ctx, userKey, "last_trained", time.Now().Unix())

	// Execute pipeline
	_, err := pipe.Exec(rbf.ctx)
	if err != nil {
		return fmt.Errorf("training failed: %v", err)
	}

	return nil
}

// ClassifyText returns spam probability for the given text
func (rbf *RedisBayesianFilter) ClassifyText(subject, body, user string) (float64, error) {
	if user == "" {
		user = rbf.config.DefaultUser
	}

	// Check if we have enough training data
	userStats, err := rbf.GetUserStats(user)
	if err != nil {
		return 0.5, err
	}

	if userStats.SpamLearned < rbf.config.MinLearns || userStats.HamLearned < rbf.config.MinLearns {
		return 0.5, nil // Not enough training data
	}

	// Generate tokens
	text := subject + " " + body
	tokens := rbf.tokenizer.GenerateOSBTokens(text)

	if len(tokens) == 0 {
		return 0.5, nil
	}

	// Get token statistics using pipeline
	pipe := rbf.client.Pipeline()
	tokenCmds := make([]*redis.MapStringStringCmd, len(tokens))

	for i, token := range tokens {
		tokenKey := rbf.getTokenKey(user, token)
		tokenCmds[i] = pipe.HGetAll(rbf.ctx, tokenKey)
	}

	_, err = pipe.Exec(rbf.ctx)
	if err != nil {
		return 0.5, fmt.Errorf("failed to get token stats: %v", err)
	}

	// Calculate spam probability using Robinson's method (like Rspamd)
	var probs []float64

	for _, cmd := range tokenCmds {
		tokenStats := cmd.Val()
		if len(tokenStats) == 0 {
			continue // Token not seen before
		}

		spamCount, _ := strconv.Atoi(tokenStats["spam"])
		hamCount, _ := strconv.Atoi(tokenStats["ham"])

		if spamCount == 0 && hamCount == 0 {
			continue
		}

		// Calculate token probability with Laplace smoothing
		spamProb := float64(spamCount+1) / float64(userStats.SpamTokens+2)
		hamProb := float64(hamCount+1) / float64(userStats.HamTokens+2)

		// Token spamminess
		tokenSpaminess := spamProb / (spamProb + hamProb)

		// Weight by how significant this token is
		significance := math.Abs(tokenSpaminess - 0.5)
		if significance > 0.1 { // Only use significant tokens
			probs = append(probs, tokenSpaminess)
		}
	}

	if len(probs) == 0 {
		return 0.5, nil
	}

	// Sort probabilities and take the most significant ones
	sort.Float64s(probs)
	maxProbs := 15 // Like Rspamd
	if len(probs) > maxProbs {
		// Take most extreme probabilities
		var significant []float64
		significant = append(significant, probs[:maxProbs/2]...)
		significant = append(significant, probs[len(probs)-maxProbs/2:]...)
		probs = significant
	}

	// Robinson's geometric mean method
	spamProduct := 1.0
	hamProduct := 1.0

	for _, prob := range probs {
		spamProduct *= prob
		hamProduct *= (1.0 - prob)
	}

	n := float64(len(probs))
	spamGeom := math.Pow(spamProduct, 1.0/n)
	hamGeom := math.Pow(hamProduct, 1.0/n)

	// Final probability
	finalProb := spamGeom / (spamGeom + hamGeom)

	return finalProb, nil
}

// UserStats contains per-user statistics
type UserStats struct {
	User        string    `json:"user"`
	SpamLearned int       `json:"spam_learned"`
	HamLearned  int       `json:"ham_learned"`
	SpamTokens  int       `json:"spam_tokens"`
	HamTokens   int       `json:"ham_tokens"`
	LastTrained time.Time `json:"last_trained"`
}

// GetUserStats returns statistics for a user
func (rbf *RedisBayesianFilter) GetUserStats(user string) (*UserStats, error) {
	if user == "" {
		user = rbf.config.DefaultUser
	}

	userKey := rbf.getUserKey(user)
	stats := rbf.client.HGetAll(rbf.ctx, userKey).Val()

	spamLearned, _ := strconv.Atoi(stats["spam_learned"])
	hamLearned, _ := strconv.Atoi(stats["ham_learned"])
	spamTokens, _ := strconv.Atoi(stats["spam_tokens"])
	hamTokens, _ := strconv.Atoi(stats["ham_tokens"])
	lastTrained, _ := strconv.ParseInt(stats["last_trained"], 10, 64)

	return &UserStats{
		User:        user,
		SpamLearned: spamLearned,
		HamLearned:  hamLearned,
		SpamTokens:  spamTokens,
		HamTokens:   hamTokens,
		LastTrained: time.Unix(lastTrained, 0),
	}, nil
}

// GetTopTokens returns the most significant spam/ham tokens
func (rbf *RedisBayesianFilter) GetTopTokens(user string, isSpam bool, limit int) ([]*TokenStats, error) {
	if user == "" {
		user = rbf.config.DefaultUser
	}

	// Scan for tokens (this could be optimized with a separate index)
	pattern := rbf.getTokenKey(user, "*")

	var tokens []*TokenStats
	iter := rbf.client.Scan(rbf.ctx, 0, pattern, int64(limit*10)).Iterator()

	for iter.Next(rbf.ctx) {
		key := iter.Val()
		tokenStats := rbf.client.HGetAll(rbf.ctx, key).Val()

		spamCount, _ := strconv.Atoi(tokenStats["spam"])
		hamCount, _ := strconv.Atoi(tokenStats["ham"])

		if spamCount > 0 || hamCount > 0 {
			// Extract token from key
			parts := strings.Split(key, ":")
			token := parts[len(parts)-1]

			// Calculate spamminess
			total := spamCount + hamCount
			spamminess := float64(spamCount) / float64(total)

			tokens = append(tokens, &TokenStats{
				Token:      token,
				SpamCount:  spamCount,
				HamCount:   hamCount,
				Spamminess: spamminess,
			})
		}
	}

	// Sort by spamminess
	if isSpam {
		sort.Slice(tokens, func(i, j int) bool {
			return tokens[i].Spamminess > tokens[j].Spamminess
		})
	} else {
		sort.Slice(tokens, func(i, j int) bool {
			return tokens[i].Spamminess < tokens[j].Spamminess
		})
	}

	if len(tokens) > limit {
		tokens = tokens[:limit]
	}

	return tokens, nil
}

// TokenStats contains token statistics
type TokenStats struct {
	Token      string  `json:"token"`
	SpamCount  int     `json:"spam_count"`
	HamCount   int     `json:"ham_count"`
	Spamminess float64 `json:"spamminess"`
}

// Helper methods
func (rbf *RedisBayesianFilter) getUserKey(user string) string {
	return fmt.Sprintf("%s:user:%s", rbf.config.KeyPrefix, user)
}

func (rbf *RedisBayesianFilter) getTokenKey(user, token string) string {
	// Hash long tokens to keep key size manageable
	if len(token) > 64 {
		h := sha1.Sum([]byte(token))
		token = fmt.Sprintf("hash_%x", h)
	}
	return fmt.Sprintf("%s:token:%s:%s", rbf.config.KeyPrefix, user, token)
}

// startCleanupRoutine starts the token cleanup routine
func (rbf *RedisBayesianFilter) startCleanupRoutine() {
	ticker := time.NewTicker(rbf.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rbf.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens removes expired tokens
func (rbf *RedisBayesianFilter) cleanupExpiredTokens() {
	// This would implement token cleanup logic
	// For now, Redis TTL handles expiration automatically
}

// Close closes the Redis connection
func (rbf *RedisBayesianFilter) Close() error {
	return rbf.client.Close()
}

// Reset clears all training data for a user
func (rbf *RedisBayesianFilter) Reset(user string) error {
	if user == "" {
		user = rbf.config.DefaultUser
	}

	// Delete user stats
	userKey := rbf.getUserKey(user)
	if err := rbf.client.Del(rbf.ctx, userKey).Err(); err != nil {
		return err
	}

	// Delete all tokens for this user
	pattern := rbf.getTokenKey(user, "*")
	iter := rbf.client.Scan(rbf.ctx, 0, pattern, 1000).Iterator()

	pipe := rbf.client.Pipeline()
	count := 0

	for iter.Next(rbf.ctx) {
		pipe.Del(rbf.ctx, iter.Val())
		count++

		// Execute in batches
		if count >= rbf.config.BatchSize {
			pipe.Exec(rbf.ctx)
			pipe = rbf.client.Pipeline()
			count = 0
		}
	}

	if count > 0 {
		_, err := pipe.Exec(rbf.ctx)
		return err
	}

	return nil
}
