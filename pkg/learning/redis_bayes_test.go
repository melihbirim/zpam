package learning

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestRedisConfig for testing
var testRedisConfig = &RedisConfig{
	RedisURL:        "redis://localhost:6379",
	KeyPrefix:       "zpo:test:bayes",
	DatabaseNum:     1, // Use separate database for testing
	OSBWindowSize:   5,
	MinTokenLength:  3,
	MaxTokenLength:  32,
	MaxTokens:       1000,
	MinLearns:       10, // Lower for testing
	MaxLearns:       5000,
	SpamThreshold:   0.95,
	PerUserStats:    true,
	DefaultUser:     "global",
	TokenTTL:        time.Hour,
	CleanupInterval: 30 * time.Minute,
	LocalCache:      false, // Disable for testing
	CacheTTL:        5 * time.Minute,
	BatchSize:       100,
}

func TestNewRedisBayesianFilter(t *testing.T) {
	// Skip if Redis not available
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer rbf.Close()

	if rbf.client == nil {
		t.Error("Redis client should not be nil")
	}
	if rbf.tokenizer == nil {
		t.Error("Tokenizer should not be nil")
	}
}

func TestOSBTokenizer(t *testing.T) {
	tokenizer := &OSBTokenizer{config: testRedisConfig}

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Simple text",
			text:     "hello world",
			expected: []string{"hello", "world", "hello|world|1"},
		},
		{
			name:     "Spam-like text",
			text:     "BUY NOW! FREE OFFER!!!",
			expected: []string{"buy", "now", "free", "offer", "buy|now|1", "buy|free|2", "buy|offer|3", "now|free|1", "now|offer|2", "free|offer|1"},
		},
		{
			name:     "Empty text",
			text:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tokenizer.GenerateOSBTokens(tt.text)

			// Check if expected tokens are present (order might vary)
			for _, expectedToken := range tt.expected {
				found := false
				for _, token := range tokens {
					if token == expectedToken {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected token '%s' not found in %v", expectedToken, tokens)
				}
			}
		})
	}
}

func TestRedisBayesianTraining(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("testuser") // Clean up
		rbf.Close()
	}()

	// Train on spam
	err = rbf.TrainSpam("BUY NOW! Free offer!", "Click here to claim your FREE prize! Act now!", "testuser")
	if err != nil {
		t.Fatalf("Failed to train spam: %v", err)
	}

	// Train on ham
	err = rbf.TrainHam("Meeting tomorrow", "Hi, let's meet tomorrow at 3pm to discuss the project.", "testuser")
	if err != nil {
		t.Fatalf("Failed to train ham: %v", err)
	}

	// Check user stats
	stats, err := rbf.GetUserStats("testuser")
	if err != nil {
		t.Fatalf("Failed to get user stats: %v", err)
	}

	if stats.SpamLearned != 1 {
		t.Errorf("Expected 1 spam learned, got %d", stats.SpamLearned)
	}
	if stats.HamLearned != 1 {
		t.Errorf("Expected 1 ham learned, got %d", stats.HamLearned)
	}
}

func TestRedisBayesianClassification(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("testuser")
		rbf.Close()
	}()

	// Train with enough samples
	spamSamples := []struct{ subject, body string }{
		{"FREE MONEY NOW!", "Click here to get FREE money! Limited time offer!"},
		{"WIN BIG PRIZE!", "You have won a big prize! Claim now!"},
		{"URGENT ACTION REQUIRED", "Act now to claim your lottery winnings!"},
		{"VIAGRA CHEAP!", "Buy cheap viagra online now!"},
		{"LOAN APPROVED!", "Your loan has been approved! Click here!"},
		{"WEIGHT LOSS MIRACLE", "Lose weight fast with this miracle drug!"},
		{"CASINO BONUS", "Get free casino bonus! Play now!"},
		{"INHERITANCE CLAIM", "You have inherited money from Nigeria!"},
		{"CREDIT REPAIR", "Repair your credit instantly! No questions asked!"},
		{"MAKE MONEY FAST", "Make $5000 per week working from home!"},
		{"ROLEX REPLICA", "Buy cheap Rolex replicas! Perfect quality!"},
		{"DEBT CONSOLIDATION", "Consolidate all your debts now!"},
	}

	hamSamples := []struct{ subject, body string }{
		{"Meeting tomorrow", "Hi, let's meet tomorrow at 3pm to discuss the project."},
		{"Weekly report", "Please find attached the weekly sales report for review."},
		{"Birthday party", "You're invited to my birthday party this Saturday!"},
		{"Project update", "The project is progressing well and on schedule."},
		{"Lunch plans", "Would you like to grab lunch together today?"},
		{"Conference call", "We have a conference call scheduled for 2pm."},
		{"Document review", "Could you please review the attached document?"},
		{"Team meeting", "Our team meeting has been moved to Friday."},
		{"Travel arrangements", "Your flight details have been confirmed."},
		{"Holiday schedule", "Please note the office holiday schedule."},
		{"Training session", "Mandatory training session next week."},
		{"System maintenance", "Scheduled system maintenance this weekend."},
	}

	// Train spam samples
	for _, sample := range spamSamples {
		err = rbf.TrainSpam(sample.subject, sample.body, "testuser")
		if err != nil {
			t.Fatalf("Failed to train spam: %v", err)
		}
	}

	// Train ham samples
	for _, sample := range hamSamples {
		err = rbf.TrainHam(sample.subject, sample.body, "testuser")
		if err != nil {
			t.Fatalf("Failed to train ham: %v", err)
		}
	}

	// Test classification
	tests := []struct {
		name       string
		subject    string
		body       string
		expectSpam bool
	}{
		{
			name:       "Clear spam",
			subject:    "FREE MONEY URGENT!",
			body:       "Click here NOW to get FREE money! Limited time!",
			expectSpam: true,
		},
		{
			name:       "Clear ham",
			subject:    "Quarterly report",
			body:       "Please review the quarterly financial report.",
			expectSpam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob, err := rbf.ClassifyText(tt.subject, tt.body, "testuser")
			if err != nil {
				t.Fatalf("Classification failed: %v", err)
			}

			isSpam := prob > 0.5
			if isSpam != tt.expectSpam {
				t.Errorf("Expected spam=%v, got spam=%v (prob=%.3f)", tt.expectSpam, isSpam, prob)
			}

			t.Logf("Text: '%s %s' -> Probability: %.3f, IsSpam: %v", tt.subject, tt.body, prob, isSpam)
		})
	}
}

func TestMultiUserSupport(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("user1")
		rbf.Reset("user2")
		rbf.Close()
	}()

	// Train different users differently
	err = rbf.TrainSpam("SPAM!", "This is spam!", "user1")
	if err != nil {
		t.Fatalf("Failed to train user1: %v", err)
	}

	err = rbf.TrainHam("HAM!", "This is ham!", "user2")
	if err != nil {
		t.Fatalf("Failed to train user2: %v", err)
	}

	// Check stats are separate
	stats1, err := rbf.GetUserStats("user1")
	if err != nil {
		t.Fatalf("Failed to get user1 stats: %v", err)
	}

	stats2, err := rbf.GetUserStats("user2")
	if err != nil {
		t.Fatalf("Failed to get user2 stats: %v", err)
	}

	if stats1.SpamLearned != 1 || stats1.HamLearned != 0 {
		t.Errorf("User1: Expected 1 spam, 0 ham, got %d spam, %d ham", stats1.SpamLearned, stats1.HamLearned)
	}

	if stats2.SpamLearned != 0 || stats2.HamLearned != 1 {
		t.Errorf("User2: Expected 0 spam, 1 ham, got %d spam, %d ham", stats2.SpamLearned, stats2.HamLearned)
	}
}

func TestTopTokens(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("testuser")
		rbf.Close()
	}()

	// Train with distinctive tokens
	for i := 0; i < 5; i++ {
		err = rbf.TrainSpam("spam token here", "This contains spam token multiple times", "testuser")
		if err != nil {
			t.Fatalf("Failed to train spam: %v", err)
		}
	}

	for i := 0; i < 5; i++ {
		err = rbf.TrainHam("ham token here", "This contains ham token multiple times", "testuser")
		if err != nil {
			t.Fatalf("Failed to train ham: %v", err)
		}
	}

	// Get top spam tokens
	spamTokens, err := rbf.GetTopTokens("testuser", true, 10)
	if err != nil {
		t.Fatalf("Failed to get top spam tokens: %v", err)
	}

	// Get top ham tokens
	hamTokens, err := rbf.GetTopTokens("testuser", false, 10)
	if err != nil {
		t.Fatalf("Failed to get top ham tokens: %v", err)
	}

	t.Logf("Top spam tokens: %d found", len(spamTokens))
	for i, token := range spamTokens {
		if i < 3 { // Log first 3
			t.Logf("Spam token: %s (spam: %d, ham: %d, spamminess: %.3f)",
				token.Token, token.SpamCount, token.HamCount, token.Spamminess)
		}
	}

	t.Logf("Top ham tokens: %d found", len(hamTokens))
	for i, token := range hamTokens {
		if i < 3 { // Log first 3
			t.Logf("Ham token: %s (spam: %d, ham: %d, spamminess: %.3f)",
				token.Token, token.SpamCount, token.HamCount, token.Spamminess)
		}
	}
}

func TestReset(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping test")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		t.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer rbf.Close()

	// Train some data
	err = rbf.TrainSpam("test", "test", "testuser")
	if err != nil {
		t.Fatalf("Failed to train: %v", err)
	}

	// Check data exists
	stats, err := rbf.GetUserStats("testuser")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.SpamLearned == 0 {
		t.Error("Expected training data to exist")
	}

	// Reset
	err = rbf.Reset("testuser")
	if err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	// Check data is gone
	stats, err = rbf.GetUserStats("testuser")
	if err != nil {
		t.Fatalf("Failed to get stats after reset: %v", err)
	}
	if stats.SpamLearned != 0 || stats.HamLearned != 0 {
		t.Errorf("Expected no training data after reset, got spam=%d, ham=%d", stats.SpamLearned, stats.HamLearned)
	}
}

// Helper function to check if Redis is available
func isRedisAvailable() bool {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use test database
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	return err == nil
}

// Benchmark tests
func BenchmarkOSBTokenization(b *testing.B) {
	tokenizer := &OSBTokenizer{config: testRedisConfig}
	text := "This is a sample email body with multiple words that should be tokenized efficiently using OSB algorithm for spam detection purposes."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tokenizer.GenerateOSBTokens(text)
	}
}

func BenchmarkRedisTraining(b *testing.B) {
	if !isRedisAvailable() {
		b.Skip("Redis not available, skipping benchmark")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		b.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("benchuser")
		rbf.Close()
	}()

	subject := "Test subject"
	body := "This is a test email body for benchmarking training performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			rbf.TrainSpam(subject, body, "benchuser")
		} else {
			rbf.TrainHam(subject, body, "benchuser")
		}
	}
}

func BenchmarkRedisClassification(b *testing.B) {
	if !isRedisAvailable() {
		b.Skip("Redis not available, skipping benchmark")
	}

	rbf, err := NewRedisBayesianFilter(testRedisConfig)
	if err != nil {
		b.Fatalf("Failed to create Redis Bayesian filter: %v", err)
	}
	defer func() {
		rbf.Reset("benchuser")
		rbf.Close()
	}()

	// Pre-train some data
	for i := 0; i < 100; i++ {
		rbf.TrainSpam("spam subject", "spam body content", "benchuser")
		rbf.TrainHam("ham subject", "ham body content", "benchuser")
	}

	subject := "Test classification subject"
	body := "This is a test email body for benchmarking classification performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rbf.ClassifyText(subject, body, "benchuser")
	}
}
