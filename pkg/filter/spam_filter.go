package filter

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zpam/spam-filter/pkg/config"
	"github.com/zpam/spam-filter/pkg/email"
	"github.com/zpam/spam-filter/pkg/headers"
	"github.com/zpam/spam-filter/pkg/learning"
	"github.com/zpam/spam-filter/pkg/plugins"
	"github.com/zpam/spam-filter/pkg/tracker"
)

// FilterResults contains the results of email filtering
type FilterResults struct {
	Total int
	Spam  int
	Ham   int
}

// SpamFilter implements the ZPAM spam detection algorithm
type SpamFilter struct {
	parser        *email.Parser
	config        *config.Config
	tracker       *tracker.FrequencyTracker
	learner       learning.BayesianClassifier // Use interface instead of concrete type
	validator     *headers.Validator
	pluginManager *plugins.DefaultPluginManager

	// Legacy fields for backward compatibility
	keywords SpamKeywords
	weights  FeatureWeights
}

// SpamKeywords contains keyword patterns for spam detection
type SpamKeywords struct {
	HighRisk   []string // Keywords that strongly indicate spam
	MediumRisk []string // Keywords that moderately indicate spam
	LowRisk    []string // Keywords that slightly indicate spam
}

// FeatureWeights defines the weights for different spam detection features
type FeatureWeights struct {
	// Content weights
	SubjectKeywords  float64
	BodyKeywords     float64
	CapsRatio        float64
	ExclamationRatio float64
	URLDensity       float64
	HTMLRatio        float64

	// Technical weights
	SuspiciousHeaders float64
	AttachmentRisk    float64
	DomainReputation  float64
	EncodingIssues    float64

	// Behavioral weights
	FromToMismatch   float64
	SubjectLength    float64
	FrequencyPenalty float64

	// Learning weights
	WordFrequency float64

	// Header validation weights
	HeaderValidation float64
}

// NewSpamFilter creates a new spam filter with default configuration
func NewSpamFilter() *SpamFilter {
	cfg := config.DefaultConfig()
	return NewSpamFilterWithConfig(cfg)
}

// NewSpamFilterWithConfig creates a new spam filter with custom configuration
func NewSpamFilterWithConfig(cfg *config.Config) *SpamFilter {
	sf := &SpamFilter{
		parser:        email.NewParser(),
		config:        cfg,
		tracker:       tracker.NewFrequencyTracker(60, cfg.Performance.CacheSize), // 60 minute window
		keywords:      convertConfigKeywords(cfg.Detection.Keywords),
		weights:       convertConfigWeights(cfg.Detection.Weights),
		pluginManager: plugins.NewPluginManager(),
	}

	// Initialize headers validator
	if cfg.Headers.EnableSPF || cfg.Headers.EnableDKIM || cfg.Headers.EnableDMARC {
		headersConfig := &headers.Config{
			EnableSPF:             cfg.Headers.EnableSPF,
			EnableDKIM:            cfg.Headers.EnableDKIM,
			EnableDMARC:           cfg.Headers.EnableDMARC,
			DNSTimeout:            time.Duration(cfg.Headers.DNSTimeoutMs) * time.Millisecond,
			MaxHopCount:           cfg.Headers.MaxHopCount,
			SuspiciousServerScore: cfg.Headers.SuspiciousServerScore,
			CacheSize:             cfg.Headers.CacheSize,
			CacheTTL:              time.Duration(cfg.Headers.CacheTTLMin) * time.Minute,
		}

		// Use default suspicious patterns
		headersConfig.SuspiciousServers = []string{
			"suspicious", "spam", "bulk", "mass", "marketing",
			"promo", "offer", "deal", "free", "win",
		}
		headersConfig.OpenRelayPatterns = []string{
			"unknown", "dynamic", "dhcp", "dial", "cable",
			"dsl", "adsl", "pool", "client", "user",
		}

		sf.validator = headers.NewValidator(headersConfig)
	}

	// Initialize Bayesian learner if enabled
	if cfg.Learning.Enabled {
		switch cfg.Learning.Backend {
		case "redis":
			// Convert duration strings to time.Duration
			tokenTTL, _ := time.ParseDuration(cfg.Learning.Redis.TokenTTL)
			cleanupInterval, _ := time.ParseDuration(cfg.Learning.Redis.CleanupInterval)
			cacheTTL, _ := time.ParseDuration(cfg.Learning.Redis.CacheTTL)

			redisConfig := &learning.RedisConfig{
				RedisURL:        cfg.Learning.Redis.RedisURL,
				KeyPrefix:       cfg.Learning.Redis.KeyPrefix,
				DatabaseNum:     cfg.Learning.Redis.DatabaseNum,
				OSBWindowSize:   cfg.Learning.Redis.OSBWindowSize,
				MinTokenLength:  cfg.Learning.Redis.MinTokenLength,
				MaxTokenLength:  cfg.Learning.Redis.MaxTokenLength,
				MaxTokens:       cfg.Learning.Redis.MaxTokens,
				MinLearns:       cfg.Learning.Redis.MinLearns,
				MaxLearns:       cfg.Learning.Redis.MaxLearns,
				SpamThreshold:   cfg.Learning.Redis.SpamThreshold,
				PerUserStats:    cfg.Learning.Redis.PerUserStats,
				DefaultUser:     cfg.Learning.Redis.DefaultUser,
				TokenTTL:        tokenTTL,
				CleanupInterval: cleanupInterval,
				LocalCache:      cfg.Learning.Redis.LocalCache,
				CacheTTL:        cacheTTL,
				BatchSize:       cfg.Learning.Redis.BatchSize,
			}

			redisFilter, err := learning.NewRedisBayesianFilter(redisConfig)
			if err != nil {
				fmt.Printf("Warning: Failed to initialize Redis Bayesian filter: %v\n", err)
				fmt.Printf("Falling back to file-based learning\n")
				// Fallback to file-based learning
				fileConfig := &learning.Config{
					MinWordLength:     cfg.Learning.File.MinWordLength,
					MaxWordLength:     cfg.Learning.File.MaxWordLength,
					CaseSensitive:     cfg.Learning.File.CaseSensitive,
					SpamThreshold:     cfg.Learning.File.SpamThreshold,
					MinWordCount:      cfg.Learning.File.MinWordCount,
					SmoothingFactor:   cfg.Learning.File.SmoothingFactor,
					UseSubjectWords:   cfg.Learning.File.UseSubjectWords,
					UseBodyWords:      cfg.Learning.File.UseBodyWords,
					UseHeaderWords:    cfg.Learning.File.UseHeaderWords,
					MaxVocabularySize: cfg.Learning.File.MaxVocabularySize,
				}
				fileFilter := learning.NewWordFrequency(fileConfig)
				sf.learner = learning.NewWordFrequencyAdapter(fileFilter)
			} else {
				sf.learner = redisFilter
			}

		case "file":
			fallthrough
		default:
			// File-based learning
			fileConfig := &learning.Config{
				MinWordLength:     cfg.Learning.File.MinWordLength,
				MaxWordLength:     cfg.Learning.File.MaxWordLength,
				CaseSensitive:     cfg.Learning.File.CaseSensitive,
				SpamThreshold:     cfg.Learning.File.SpamThreshold,
				MinWordCount:      cfg.Learning.File.MinWordCount,
				SmoothingFactor:   cfg.Learning.File.SmoothingFactor,
				UseSubjectWords:   cfg.Learning.File.UseSubjectWords,
				UseBodyWords:      cfg.Learning.File.UseBodyWords,
				UseHeaderWords:    cfg.Learning.File.UseHeaderWords,
				MaxVocabularySize: cfg.Learning.File.MaxVocabularySize,
			}

			fileFilter := learning.NewWordFrequency(fileConfig)
			sf.learner = learning.NewWordFrequencyAdapter(fileFilter)

			// Try to load existing model for file-based learning
			if adapter, ok := sf.learner.(*learning.WordFrequencyAdapter); ok {
				if _, err := os.Stat(cfg.Learning.File.ModelPath); err == nil {
					if err := adapter.WordFrequency.LoadModel(cfg.Learning.File.ModelPath); err != nil {
						fmt.Printf("Warning: Failed to load learning model: %v\n", err)
					}
				}
			}
		}
	}

	// Initialize plugins
	sf.initializePlugins()

	return sf
}

// LoadConfigFromPath loads configuration from file path or returns default
func LoadConfigFromPath(configPath string) (*config.Config, error) {
	if configPath == "" {
		return config.DefaultConfig(), nil
	}

	return config.LoadConfig(configPath)
}

// initializePlugins sets up and configures the plugin system
func (sf *SpamFilter) initializePlugins() {
	if sf.pluginManager == nil {
		return
	}

	// Register built-in plugins
	sf.pluginManager.RegisterPlugin(plugins.NewSpamAssassinPlugin())
	sf.pluginManager.RegisterPlugin(plugins.NewRspamdPlugin())
	sf.pluginManager.RegisterPlugin(plugins.NewCustomRulesPlugin())
	sf.pluginManager.RegisterPlugin(plugins.NewVirusTotalPlugin())
	sf.pluginManager.RegisterPlugin(plugins.NewMLPlugin())

	// Load plugin configurations if available
	if sf.config != nil && sf.config.Plugins.Enabled {
		pluginConfigs := map[string]*plugins.PluginConfig{
			"spamassassin": convertConfigToPluginConfig(sf.config.Plugins.SpamAssassin),
			"rspamd":       convertConfigToPluginConfig(sf.config.Plugins.Rspamd),
			"custom_rules": convertConfigToPluginConfig(sf.config.Plugins.CustomRules),
		}

		if err := sf.pluginManager.LoadPlugins(pluginConfigs); err != nil {
			fmt.Printf("Warning: Failed to load plugins: %v\n", err)
		}
	}
}

// convertConfigToPluginConfig converts config.PluginConfig to plugins.PluginConfig
func convertConfigToPluginConfig(cfg config.PluginConfig) *plugins.PluginConfig {
	return &plugins.PluginConfig{
		Enabled:  cfg.Enabled,
		Weight:   cfg.Weight,
		Priority: cfg.Priority,
		Timeout:  time.Duration(cfg.Timeout) * time.Millisecond,
		Settings: cfg.Settings,
	}
}

// TestEmail tests a single email and returns spam score (1-5)
func (sf *SpamFilter) TestEmail(filepath string) (int, error) {
	email, err := sf.parser.ParseFromFile(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to parse email: %v", err)
	}

	score := sf.calculateSpamScore(email)
	return sf.normalizeScore(score), nil
}

// CalculateSpamScore calculates the raw spam score for an email (public method for milter integration)
func (sf *SpamFilter) CalculateSpamScore(email *email.Email) float64 {
	return sf.calculateSpamScore(email)
}

// NormalizeScore converts raw score to 1-5 scale (public method for milter integration)
func (sf *SpamFilter) NormalizeScore(rawScore float64) int {
	return sf.normalizeScore(rawScore)
}

// ProcessEmails processes a directory of emails and filters spam
func (sf *SpamFilter) ProcessEmails(inputPath, outputPath, spamPath string, threshold int) (*FilterResults, error) {
	results := &FilterResults{}

	// Create output directories if they don't exist
	if outputPath != "" {
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	if spamPath != "" {
		if err := os.MkdirAll(spamPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create spam directory: %v", err)
		}
	}

	// Collect all email files first
	var emailFiles []string
	err := filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-email files
		if d.IsDir() || !sf.isEmailFile(path) {
			return nil
		}

		emailFiles = append(emailFiles, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(emailFiles) == 0 {
		return results, nil
	}

	// Get concurrency setting from config
	maxConcurrent := 20 // Default
	if sf.config != nil && sf.config.Performance.MaxConcurrentEmails > 0 {
		maxConcurrent = sf.config.Performance.MaxConcurrentEmails
	}

	// Process emails in parallel
	return sf.processEmailsParallel(emailFiles, outputPath, spamPath, threshold, maxConcurrent)
}

// processEmailsParallel processes emails using parallel worker goroutines
func (sf *SpamFilter) processEmailsParallel(emailFiles []string, outputPath, spamPath string, threshold, maxConcurrent int) (*FilterResults, error) {
	// Results tracking with atomic operations for thread safety
	var totalProcessed, spamDetected, hamDetected int32
	var processingErrors int32

	// Create worker pool
	type EmailJob struct {
		FilePath string
		Index    int
	}

	type EmailResult struct {
		FilePath    string
		Score       int
		IsSpam      bool
		Error       error
		ProcessTime time.Duration
	}

	// Channels for work distribution
	jobChan := make(chan EmailJob, len(emailFiles))
	resultChan := make(chan EmailResult, len(emailFiles))

	// Worker pool with goroutines
	var workerWG sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < maxConcurrent; i++ {
		workerWG.Add(1)
		go func(workerID int) {
			defer workerWG.Done()

			for job := range jobChan {
				startTime := time.Now()

				// Process single email
				score, err := sf.TestEmail(job.FilePath)
				processingTime := time.Since(startTime)

				// Send result
				resultChan <- EmailResult{
					FilePath:    job.FilePath,
					Score:       score,
					IsSpam:      score >= threshold,
					Error:       err,
					ProcessTime: processingTime,
				}

				// Update counters atomically
				atomic.AddInt32(&totalProcessed, 1)

				if err != nil {
					atomic.AddInt32(&processingErrors, 1)
				} else if score >= threshold {
					atomic.AddInt32(&spamDetected, 1)
				} else {
					atomic.AddInt32(&hamDetected, 1)
				}
			}
		}(i)
	}

	// Send jobs to workers
	go func() {
		defer close(jobChan)
		for i, filePath := range emailFiles {
			jobChan <- EmailJob{
				FilePath: filePath,
				Index:    i,
			}
		}
	}()

	// Collect results and handle file moving
	go func() {
		defer close(resultChan)
		workerWG.Wait()
	}()

	// Process results as they come in (file moving can also be parallel)
	var moveWG sync.WaitGroup
	var moveErrors int32

	for result := range resultChan {
		if result.Error != nil {
			fmt.Printf("Warning: Failed to process %s: %v\n", result.FilePath, result.Error)
			continue
		}

		// Move file in parallel (non-blocking)
		moveWG.Add(1)
		go func(res EmailResult) {
			defer moveWG.Done()

			var destPath string
			if res.IsSpam && spamPath != "" {
				destPath = filepath.Join(spamPath, filepath.Base(res.FilePath))
			} else if !res.IsSpam && outputPath != "" {
				destPath = filepath.Join(outputPath, filepath.Base(res.FilePath))
			}

			if destPath != "" {
				if err := sf.moveFile(res.FilePath, destPath); err != nil {
					fmt.Printf("Warning: Failed to move %s: %v\n", res.FilePath, err)
					atomic.AddInt32(&moveErrors, 1)
				}
			}
		}(result)
	}

	// Wait for all file moves to complete
	moveWG.Wait()

	// Build final results
	finalResults := &FilterResults{
		Total: int(atomic.LoadInt32(&totalProcessed)),
		Spam:  int(atomic.LoadInt32(&spamDetected)),
		Ham:   int(atomic.LoadInt32(&hamDetected)),
	}

	// Report any errors
	if errors := atomic.LoadInt32(&processingErrors); errors > 0 {
		fmt.Printf("Warning: %d emails failed to process\n", errors)
	}
	if moveErr := atomic.LoadInt32(&moveErrors); moveErr > 0 {
		fmt.Printf("Warning: %d emails failed to move\n", moveErr)
	}

	return finalResults, nil
}

// calculateSpamScore calculates raw spam score for an email
func (sf *SpamFilter) calculateSpamScore(email *email.Email) float64 {
	// Check whitelist/blacklist first
	domain := sf.extractDomain(email.From)
	if sf.config != nil {
		if sf.config.IsWhitelisted(email.From, domain) {
			return 0 // Whitelisted emails are never spam
		}
		if sf.config.IsBlacklisted(email.From, domain) {
			return 25 // Blacklisted emails are always spam
		}
	}

	// Parallel feature scoring for maximum performance
	type FeatureScore struct {
		Name  string
		Score float64
	}

	// Create channels for parallel feature calculation
	scoreChan := make(chan FeatureScore, 12) // Buffer for all features
	var wg sync.WaitGroup

	// Content-based scoring (parallel)
	if sf.config == nil || sf.config.Detection.Features.KeywordDetection {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreKeywords(email.Subject, email.Body)
			scoreChan <- FeatureScore{"keywords", score}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreCapsRatio(email.Features.SubjectCapsRatio, email.Features.BodyCapsRatio)
		scoreChan <- FeatureScore{"caps", score}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreExclamations(email.Features.SubjectExclamations, email.Features.BodyExclamations)
		scoreChan <- FeatureScore{"exclamations", score}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreURLs(email.Features.BodyURLCount, email.Features.BodyLength)
		scoreChan <- FeatureScore{"urls", score}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreHTML(email.Features.BodyHTMLRatio)
		scoreChan <- FeatureScore{"html", score}
	}()

	// Technical scoring (parallel)
	if sf.config == nil || sf.config.Detection.Features.HeaderAnalysis {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreSuspiciousHeaders(email.Features.SuspiciousHeaders)
			scoreChan <- FeatureScore{"suspicious_headers", score}
		}()
	}

	if sf.config == nil || sf.config.Detection.Features.AttachmentScan {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreAttachments(email.Features.AttachmentCount, email.Attachments)
			scoreChan <- FeatureScore{"attachments", score}
		}()
	}

	if sf.config == nil || sf.config.Detection.Features.DomainCheck {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreDomainReputation(email.Features.SenderDomainReputable)
			scoreChan <- FeatureScore{"domain", score}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreEncodingIssues(email.Features.EncodingIssues)
		scoreChan <- FeatureScore{"encoding", score}
	}()

	// Behavioral scoring (parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreFromToMismatch(email.Features.FromToMismatch)
		scoreChan <- FeatureScore{"from_to_mismatch", score}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		score := sf.scoreSubjectLength(email.Features.SubjectLength)
		scoreChan <- FeatureScore{"subject_length", score}
	}()

	// Word frequency learning scoring (if enabled)
	if sf.learner != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreLearning(email.Subject, email.Body)
			scoreChan <- FeatureScore{"learning", score}
		}()
	}

	// Frequency scoring (if enabled)
	if sf.config == nil || sf.config.Detection.Features.FrequencyTracking {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreFrequency(email.From, domain)
			scoreChan <- FeatureScore{"frequency", score}
		}()
	}

	// Headers validation scoring (if enabled) - this might take longer due to DNS
	if sf.validator != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			score := sf.scoreHeaders(email.Headers)
			scoreChan <- FeatureScore{"headers", score}
		}()
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(scoreChan)
	}()

	// Collect results from parallel feature scoring
	var totalScore float64
	var debugScores []string

	for featureScore := range scoreChan {
		totalScore += featureScore.Score
		debugScores = append(debugScores, fmt.Sprintf("%s:%.2f", featureScore.Name, featureScore.Score))
	}

	// Run plugins if enabled (parallel with ZPAM's native scoring)
	var pluginScore float64
	if sf.pluginManager != nil && sf.config != nil && sf.config.Plugins.Enabled {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sf.config.Plugins.Timeout)*time.Millisecond)
		defer cancel()

		pluginResults, err := sf.pluginManager.ExecuteAll(ctx, email)
		if err != nil {
			if sf.config.Logging.Level == "debug" {
				fmt.Printf("[DEBUG PLUGIN ERROR] %v\n", err)
			}
		} else {
			// Use the plugin manager's built-in score combination
			pluginScore, err = sf.pluginManager.CombineScores(pluginResults)
			if err != nil {
				if sf.config.Logging.Level == "debug" {
					fmt.Printf("[DEBUG PLUGIN COMBINE ERROR] %v\n", err)
				}
				pluginScore = 0 // Fallback to 0 if combination fails
			}

			if sf.config.Logging.Level == "debug" {
				var pluginDebug []string
				for _, result := range pluginResults {
					pluginDebug = append(pluginDebug, fmt.Sprintf("%s:%.2f", result.Name, result.Score))
				}
				fmt.Printf("[DEBUG PLUGIN SCORES] Combined:%.2f [%s]\n", pluginScore, strings.Join(pluginDebug, " "))
			}
		}
	}

	// Combine ZPAM native score with plugin score
	finalScore := totalScore + pluginScore

	// Log debug info if debugging enabled
	if sf.config != nil && sf.config.Logging.Level == "debug" {
		fmt.Printf("[DEBUG FINAL SCORE] From:%s ZPAM:%.2f Plugin:%.2f Final:%.2f [%s]\n",
			email.From, totalScore, pluginScore, finalScore, strings.Join(debugScores, " "))
	}

	return finalScore
}

// scoreKeywords scores based on spam keywords
func (sf *SpamFilter) scoreKeywords(subject, body string) float64 {
	text := strings.ToLower(subject + " " + body)
	var score float64

	// High-risk keywords
	for _, keyword := range sf.keywords.HighRisk {
		if strings.Contains(text, keyword) {
			score += sf.weights.SubjectKeywords * 3.0
		}
	}

	// Medium-risk keywords
	for _, keyword := range sf.keywords.MediumRisk {
		if strings.Contains(text, keyword) {
			score += sf.weights.BodyKeywords * 2.0
		}
	}

	// Low-risk keywords
	for _, keyword := range sf.keywords.LowRisk {
		if strings.Contains(text, keyword) {
			score += sf.weights.BodyKeywords * 1.0
		}
	}

	return score
}

// scoreCapsRatio scores based on capital letters ratio
func (sf *SpamFilter) scoreCapsRatio(subjectRatio, bodyRatio float64) float64 {
	avgRatio := (subjectRatio + bodyRatio) / 2
	if avgRatio > 0.7 {
		return sf.weights.CapsRatio * 4.0
	} else if avgRatio > 0.5 {
		return sf.weights.CapsRatio * 2.0
	} else if avgRatio > 0.3 {
		return sf.weights.CapsRatio * 1.0
	}
	return 0
}

// scoreExclamations scores based on exclamation marks
func (sf *SpamFilter) scoreExclamations(subjectExcl, bodyExcl int) float64 {
	total := subjectExcl + bodyExcl
	if total > 5 {
		return sf.weights.ExclamationRatio * 3.0
	} else if total > 3 {
		return sf.weights.ExclamationRatio * 2.0
	} else if total > 1 {
		return sf.weights.ExclamationRatio * 1.0
	}
	return 0
}

// scoreURLs scores based on URL density
func (sf *SpamFilter) scoreURLs(urlCount, bodyLength int) float64 {
	if bodyLength == 0 {
		return 0
	}

	density := float64(urlCount) / float64(bodyLength) * 1000 // URLs per 1000 chars
	if density > 10 {
		return sf.weights.URLDensity * 4.0
	} else if density > 5 {
		return sf.weights.URLDensity * 2.0
	} else if density > 2 {
		return sf.weights.URLDensity * 1.0
	}
	return 0
}

// scoreHTML scores based on HTML content ratio
func (sf *SpamFilter) scoreHTML(htmlRatio float64) float64 {
	if htmlRatio > 0.5 {
		return sf.weights.HTMLRatio * 2.0
	} else if htmlRatio > 0.2 {
		return sf.weights.HTMLRatio * 1.0
	}
	return 0
}

// scoreSuspiciousHeaders scores based on suspicious headers
func (sf *SpamFilter) scoreSuspiciousHeaders(count int) float64 {
	return sf.weights.SuspiciousHeaders * float64(count)
}

// scoreAttachments scores based on attachments
func (sf *SpamFilter) scoreAttachments(count int, attachments []email.Attachment) float64 {
	score := sf.weights.AttachmentRisk * float64(count)

	// Additional scoring for suspicious attachment types
	for _, attachment := range attachments {
		if sf.isSuspiciousAttachment(attachment) {
			score += sf.weights.AttachmentRisk * 2.0
		}
	}

	return score
}

// scoreDomainReputation scores based on sender domain reputation
func (sf *SpamFilter) scoreDomainReputation(reputable bool) float64 {
	if !reputable {
		return sf.weights.DomainReputation * 2.0
	}
	return 0
}

// scoreEncodingIssues scores based on encoding problems
func (sf *SpamFilter) scoreEncodingIssues(hasIssues bool) float64 {
	if hasIssues {
		return sf.weights.EncodingIssues * 1.5
	}
	return 0
}

// scoreFromToMismatch scores based on From/To field mismatches
func (sf *SpamFilter) scoreFromToMismatch(mismatch bool) float64 {
	if mismatch {
		return sf.weights.FromToMismatch * 2.0
	}
	return 0
}

// scoreSubjectLength scores based on subject length
func (sf *SpamFilter) scoreSubjectLength(length int) float64 {
	if length > 100 {
		return sf.weights.SubjectLength * 2.0
	} else if length < 5 {
		return sf.weights.SubjectLength * 1.5
	}
	return 0
}

// normalizeScore converts raw score to 1-5 scale
func (sf *SpamFilter) normalizeScore(rawScore float64) int {
	// Normalize to 1-5 scale based on thresholds
	if rawScore >= 20 {
		return 5 // Definitely spam
	} else if rawScore >= 15 {
		return 4 // Likely spam
	} else if rawScore >= 10 {
		return 3 // Possibly spam
	} else if rawScore >= 5 {
		return 2 // Probably clean
	} else {
		return 1 // Definitely clean
	}
}

// Helper functions

// isEmailFile checks if a file is likely an email file
func (sf *SpamFilter) isEmailFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	emailExts := []string{".eml", ".msg", ".txt", ".email"}

	for _, emailExt := range emailExts {
		if ext == emailExt {
			return true
		}
	}

	// Check if it's a file without extension (common for email files)
	return ext == ""
}

// isSuspiciousAttachment checks if an attachment is suspicious
func (sf *SpamFilter) isSuspiciousAttachment(attachment email.Attachment) bool {
	suspicious := []string{".exe", ".scr", ".bat", ".com", ".pif", ".vbs", ".js"}

	ext := strings.ToLower(filepath.Ext(attachment.Filename))
	for _, suspExt := range suspicious {
		if ext == suspExt {
			return true
		}
	}

	return false
}

// moveFile moves a file from source to destination
func (sf *SpamFilter) moveFile(src, dst string) error {
	return os.Rename(src, dst)
}

// getDefaultKeywords returns default spam keywords
func getDefaultKeywords() SpamKeywords {
	return SpamKeywords{
		HighRisk: []string{
			"free money", "get rich", "make money fast", "guaranteed income",
			"no risk", "act now", "limited time", "urgent", "congratulations",
			"you have won", "lottery", "inheritance", "nigerian prince",
			"viagra", "cialis", "pharmacy", "prescription",
		},
		MediumRisk: []string{
			"click here", "visit our website", "special offer", "discount",
			"save money", "credit", "loan", "mortgage", "insurance",
			"weight loss", "diet", "lose weight", "earn extra",
		},
		LowRisk: []string{
			"free", "offer", "deal", "sale", "promotion", "bonus",
			"gift", "prize", "winner", "selected", "opportunity",
		},
	}
}

// getOptimizedWeights returns optimized feature weights for speed
func getOptimizedWeights() FeatureWeights {
	return FeatureWeights{
		SubjectKeywords:   3.0,
		BodyKeywords:      2.0,
		CapsRatio:         1.5,
		ExclamationRatio:  1.0,
		URLDensity:        2.5,
		HTMLRatio:         1.0,
		SuspiciousHeaders: 2.0,
		AttachmentRisk:    1.5,
		DomainReputation:  3.0,
		EncodingIssues:    1.0,
		FromToMismatch:    2.0,
		SubjectLength:     0.5,
		FrequencyPenalty:  2.0,
		WordFrequency:     2.0,
		HeaderValidation:  2.5,
	}
}

// convertConfigKeywords converts config keywords to legacy format
func convertConfigKeywords(configKeywords config.KeywordLists) SpamKeywords {
	return SpamKeywords{
		HighRisk:   configKeywords.HighRisk,
		MediumRisk: configKeywords.MediumRisk,
		LowRisk:    configKeywords.LowRisk,
	}
}

// convertConfigWeights converts config weights to legacy format
func convertConfigWeights(configWeights config.FeatureWeights) FeatureWeights {
	return FeatureWeights{
		SubjectKeywords:   configWeights.SubjectKeywords,
		BodyKeywords:      configWeights.BodyKeywords,
		CapsRatio:         configWeights.CapsRatio,
		ExclamationRatio:  configWeights.ExclamationRatio,
		URLDensity:        configWeights.URLDensity,
		HTMLRatio:         configWeights.HTMLRatio,
		SuspiciousHeaders: configWeights.SuspiciousHeaders,
		AttachmentRisk:    configWeights.AttachmentRisk,
		DomainReputation:  configWeights.DomainReputation,
		EncodingIssues:    configWeights.EncodingIssues,
		FromToMismatch:    configWeights.FromToMismatch,
		SubjectLength:     configWeights.SubjectLength,
		FrequencyPenalty:  configWeights.FrequencyPenalty,
		WordFrequency:     configWeights.WordFrequency,
		HeaderValidation:  configWeights.HeaderValidation,
	}
}

// extractDomain extracts domain from email address
func (sf *SpamFilter) extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return strings.ToLower(parts[1])
	}
	return ""
}

// scoreFrequency scores based on sender frequency patterns
func (sf *SpamFilter) scoreFrequency(email, domain string) float64 {
	if sf.tracker == nil {
		return 0
	}

	// Track this sender (we don't know if it's spam yet, so pass false)
	result := sf.tracker.TrackSender(email, domain, false)

	// Apply frequency penalty weight
	weight := sf.weights.FrequencyPenalty
	if sf.config != nil {
		weight = sf.config.Detection.Weights.FrequencyPenalty
	}

	return result.FrequencyScore * weight
}

// scoreLearning scores based on learned word frequencies
func (sf *SpamFilter) scoreLearning(subject, body string) float64 {
	if sf.learner == nil {
		return 0
	}

	// Get spam probability from learner
	spamProb, _ := sf.learner.ClassifyText(subject, body, "")

	// Convert probability to score (0-1 -> 0-10)
	// Values > 0.5 are considered spammy
	if spamProb > 0.5 {
		weight := sf.weights.WordFrequency
		if sf.config != nil {
			weight = sf.config.Detection.Weights.WordFrequency
		}
		return (spamProb - 0.5) * 20 * weight // Scale to 0-10
	}

	return 0
}

// scoreHeaders scores based on email headers validation
func (sf *SpamFilter) scoreHeaders(headers map[string]string) float64 {
	if sf.validator == nil {
		return 0
	}

	// Validate headers
	result := sf.validator.ValidateHeaders(headers)

	// Calculate score based on validation results using SpamAssassin-inspired penalties
	score := 0.0

	// Get penalty values from config (SpamAssassin-inspired, much more reasonable)
	spfFailPenalty := 0.9      // Default SpamAssassin value
	dkimMissingPenalty := 1.0  // Reasonable penalty
	dmarcMissingPenalty := 1.5 // Moderate penalty
	authWeight := 1.0          // Reduced from 2.5
	suspiciousWeight := 1.0    // Reduced from 2.5

	// Use config values if available
	if sf.config != nil {
		// Check if we have SpamAssassin-inspired config values
		if sf.config.Headers.SPFFailPenalty != 0 {
			spfFailPenalty = sf.config.Headers.SPFFailPenalty
		}
		if sf.config.Headers.DKIMMissingPenalty != 0 {
			dkimMissingPenalty = sf.config.Headers.DKIMMissingPenalty
		}
		if sf.config.Headers.DMARCMissingPenalty != 0 {
			dmarcMissingPenalty = sf.config.Headers.DMARCMissingPenalty
		}
		authWeight = sf.config.Headers.AuthWeight
		suspiciousWeight = sf.config.Headers.SuspiciousWeight
	}

	// Authentication scoring (much more reasonable than before)
	authScore := result.AuthScore // 0-100
	if authScore < 50 {
		// Reduced from 0.2 to 0.05 (was adding up to 10 points, now max 2.5)
		score += (50 - authScore) * 0.05
	}

	// Suspicious score (much more reasonable)
	suspiciousScore := result.SuspiciScore // 0-100
	// Reduced from 0.15 to 0.03 (was adding up to 15 points, now max 3)
	score += suspiciousScore * 0.03

	// SPF failures (SpamAssassin-inspired penalties)
	switch result.SPF.Result {
	case "fail":
		score += spfFailPenalty // 0.9 like SpamAssassin (was 8.0!)
	case "softfail":
		score += spfFailPenalty * 0.5 // Half penalty for softfail
	case "temperror", "permerror":
		score += spfFailPenalty * 0.25 // Quarter penalty for errors
	}

	// DKIM failures (reasonable penalties)
	if !result.DKIM.Valid {
		score += dkimMissingPenalty // 1.0 (was 6.0!)
	}

	// DMARC failures (moderate penalties)
	if !result.DMARC.Valid {
		score += dmarcMissingPenalty // 1.5 (was 7.0!)
	}

	// Routing anomalies (slightly reduced)
	score += float64(len(result.Routing.SuspiciousHops)) * 2.0   // Was 3.0
	score += float64(len(result.Routing.OpenRelays)) * 2.5       // Was 4.0
	score += float64(len(result.Routing.ReverseDNSIssues)) * 1.0 // Was 2.0

	// Excessive routing hops (unchanged - this is reasonable)
	if result.Routing.HopCount > 10 {
		score += float64(result.Routing.HopCount-10) * 0.5 // Reduced from 1.0
	}

	// Header anomalies (slightly reduced)
	score += float64(len(result.Anomalies)) * 1.0 // Was 2.0

	// Apply weight from config (now much more reasonable)
	return score * authWeight * suspiciousWeight
}

// ResetLearning resets the learning model
func (sf *SpamFilter) ResetLearning(user string) error {
	if sf.learner == nil {
		return fmt.Errorf("learning is not enabled")
	}
	return sf.learner.Reset(user)
}

// TrainSpam trains the learner on spam content
func (sf *SpamFilter) TrainSpam(subject, body, user string) error {
	if sf.learner == nil {
		return fmt.Errorf("learning is not enabled")
	}
	return sf.learner.TrainSpam(subject, body, user)
}

// TrainHam trains the learner on ham content
func (sf *SpamFilter) TrainHam(subject, body, user string) error {
	if sf.learner == nil {
		return fmt.Errorf("learning is not enabled")
	}
	return sf.learner.TrainHam(subject, body, user)
}

// SaveModel saves the learning model (file backend only)
func (sf *SpamFilter) SaveModel(path string) error {
	if sf.learner == nil {
		return fmt.Errorf("learning is not enabled")
	}

	// Only file-based learning supports SaveModel
	if adapter, ok := sf.learner.(*learning.WordFrequencyAdapter); ok {
		return adapter.WordFrequency.SaveModel(path)
	}

	// For Redis backend, data is automatically persisted
	return nil
}
