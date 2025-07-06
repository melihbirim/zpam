package filter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/email"
	"github.com/zpo/spam-filter/pkg/headers"
	"github.com/zpo/spam-filter/pkg/learning"
	"github.com/zpo/spam-filter/pkg/tracker"
)

// FilterResults contains the results of email filtering
type FilterResults struct {
	Total int
	Spam  int
	Ham   int
}

// SpamFilter implements the ZPO spam detection algorithm
type SpamFilter struct {
	parser    *email.Parser
	config    *config.Config
	tracker   *tracker.FrequencyTracker
	learner   *learning.WordFrequency
	validator *headers.Validator

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
		parser:   email.NewParser(),
		config:   cfg,
		tracker:  tracker.NewFrequencyTracker(60, cfg.Performance.CacheSize), // 60 minute window
		keywords: convertConfigKeywords(cfg.Detection.Keywords),
		weights:  convertConfigWeights(cfg.Detection.Weights),
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

	// Initialize word frequency learner if enabled
	if cfg.Learning.Enabled {
		learningConfig := &learning.Config{
			MinWordLength:     cfg.Learning.MinWordLength,
			MaxWordLength:     cfg.Learning.MaxWordLength,
			CaseSensitive:     cfg.Learning.CaseSensitive,
			SpamThreshold:     cfg.Learning.SpamThreshold,
			MinWordCount:      cfg.Learning.MinWordCount,
			SmoothingFactor:   cfg.Learning.SmoothingFactor,
			UseSubjectWords:   cfg.Learning.UseSubjectWords,
			UseBodyWords:      cfg.Learning.UseBodyWords,
			UseHeaderWords:    cfg.Learning.UseHeaderWords,
			MaxVocabularySize: cfg.Learning.MaxVocabularySize,
		}

		sf.learner = learning.NewWordFrequency(learningConfig)

		// Try to load existing model
		if _, err := os.Stat(cfg.Learning.ModelPath); err == nil {
			if err := sf.learner.LoadModel(cfg.Learning.ModelPath); err != nil {
				fmt.Printf("Warning: Failed to load learning model: %v\n", err)
			}
		}
	}

	return sf
}

// LoadConfigFromPath loads configuration from file path or returns default
func LoadConfigFromPath(configPath string) (*config.Config, error) {
	if configPath == "" {
		return config.DefaultConfig(), nil
	}

	return config.LoadConfig(configPath)
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

	// Walk through input directory
	err := filepath.WalkDir(inputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-email files
		if d.IsDir() || !sf.isEmailFile(path) {
			return nil
		}

		// Process email
		score, err := sf.TestEmail(path)
		if err != nil {
			return fmt.Errorf("failed to process %s: %v", path, err)
		}

		results.Total++

		// Move email to appropriate folder
		if score >= threshold {
			results.Spam++
			if spamPath != "" {
				destPath := filepath.Join(spamPath, filepath.Base(path))
				if err := sf.moveFile(path, destPath); err != nil {
					return fmt.Errorf("failed to move spam email: %v", err)
				}
			}
		} else {
			results.Ham++
			if outputPath != "" {
				destPath := filepath.Join(outputPath, filepath.Base(path))
				if err := sf.moveFile(path, destPath); err != nil {
					return fmt.Errorf("failed to move clean email: %v", err)
				}
			}
		}

		return nil
	})

	return results, err
}

// calculateSpamScore calculates raw spam score for an email
func (sf *SpamFilter) calculateSpamScore(email *email.Email) float64 {
	var score float64

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

	// Debug: Track each feature's contribution
	var debugScores []string

	// Content-based scoring (if enabled)
	if sf.config == nil || sf.config.Detection.Features.KeywordDetection {
		keywordScore := sf.scoreKeywords(email.Subject, email.Body)
		score += keywordScore
		debugScores = append(debugScores, fmt.Sprintf("keywords:%.2f", keywordScore))
	}

	capsScore := sf.scoreCapsRatio(email.Features.SubjectCapsRatio, email.Features.BodyCapsRatio)
	score += capsScore
	debugScores = append(debugScores, fmt.Sprintf("caps:%.2f", capsScore))

	exclamScore := sf.scoreExclamations(email.Features.SubjectExclamations, email.Features.BodyExclamations)
	score += exclamScore
	debugScores = append(debugScores, fmt.Sprintf("exclamations:%.2f", exclamScore))

	urlScore := sf.scoreURLs(email.Features.BodyURLCount, email.Features.BodyLength)
	score += urlScore
	debugScores = append(debugScores, fmt.Sprintf("urls:%.2f", urlScore))

	htmlScore := sf.scoreHTML(email.Features.BodyHTMLRatio)
	score += htmlScore
	debugScores = append(debugScores, fmt.Sprintf("html:%.2f", htmlScore))

	// Technical scoring (if enabled)
	if sf.config == nil || sf.config.Detection.Features.HeaderAnalysis {
		suspiciousScore := sf.scoreSuspiciousHeaders(email.Features.SuspiciousHeaders)
		score += suspiciousScore
		debugScores = append(debugScores, fmt.Sprintf("suspicious_headers:%.2f", suspiciousScore))
	}
	if sf.config == nil || sf.config.Detection.Features.AttachmentScan {
		attachmentScore := sf.scoreAttachments(email.Features.AttachmentCount, email.Attachments)
		score += attachmentScore
		debugScores = append(debugScores, fmt.Sprintf("attachments:%.2f", attachmentScore))
	}
	if sf.config == nil || sf.config.Detection.Features.DomainCheck {
		domainScore := sf.scoreDomainReputation(email.Features.SenderDomainReputable)
		score += domainScore
		debugScores = append(debugScores, fmt.Sprintf("domain:%.2f", domainScore))
	}

	encodingScore := sf.scoreEncodingIssues(email.Features.EncodingIssues)
	score += encodingScore
	debugScores = append(debugScores, fmt.Sprintf("encoding:%.2f", encodingScore))

	// Behavioral scoring
	mismatchScore := sf.scoreFromToMismatch(email.Features.FromToMismatch)
	score += mismatchScore
	debugScores = append(debugScores, fmt.Sprintf("from_to_mismatch:%.2f", mismatchScore))

	lengthScore := sf.scoreSubjectLength(email.Features.SubjectLength)
	score += lengthScore
	debugScores = append(debugScores, fmt.Sprintf("subject_length:%.2f", lengthScore))

	// Word frequency learning scoring (if enabled)
	if sf.learner != nil {
		learningScore := sf.scoreLearning(email.Subject, email.Body)
		score += learningScore
		debugScores = append(debugScores, fmt.Sprintf("learning:%.2f", learningScore))
	}

	// Frequency scoring (if enabled)
	if sf.config == nil || sf.config.Detection.Features.FrequencyTracking {
		frequencyScore := sf.scoreFrequency(email.From, domain)
		score += frequencyScore
		debugScores = append(debugScores, fmt.Sprintf("frequency:%.2f", frequencyScore))
	}

	// Headers validation scoring (if enabled)
	if sf.validator != nil {
		headerScore := sf.scoreHeaders(email.Headers)
		score += headerScore
		debugScores = append(debugScores, fmt.Sprintf("headers:%.2f", headerScore))
	}

	// Log debug info if debugging enabled
	if sf.config != nil && sf.config.Logging.Level == "debug" {
		fmt.Printf("[DEBUG SPAM SCORE] From:%s Total:%.2f [%s]\n",
			email.From, score, strings.Join(debugScores, " "))
	}

	return score
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
	spamProb := sf.learner.ClassifyText(subject, body)

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
