package plugins

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

// MLPlugin provides machine learning-based spam detection
type MLPlugin struct {
	config   *PluginConfig
	enabled  bool
	backend  MLBackend
	stats    *MLStats
	features *FeatureExtractor
}

// MLStats tracks machine learning plugin statistics
type MLStats struct {
	ClassificationsTotal  int64   `json:"classifications_total"`
	ClassificationsFailed int64   `json:"classifications_failed"`
	SpamDetected          int64   `json:"spam_detected"`
	HamDetected           int64   `json:"ham_detected"`
	AverageConfidence     float64 `json:"average_confidence"`
	ModelLoadTime         int64   `json:"model_load_time_ms"`
	LastModelUpdate       string  `json:"last_model_update"`
}

// MLBackend interface for different ML backends
type MLBackend interface {
	Initialize(config map[string]interface{}) error
	Predict(features []float64) (*MLPrediction, error)
	GetModelInfo() (*ModelInfo, error)
	IsHealthy() error
	Cleanup() error
}

// MLPrediction represents ML model prediction result
type MLPrediction struct {
	IsSpam      bool    `json:"is_spam"`
	Confidence  float64 `json:"confidence"`
	SpamScore   float64 `json:"spam_score"`
	HamScore    float64 `json:"ham_score"`
	ModelName   string  `json:"model_name"`
	Features    int     `json:"feature_count"`
	ProcessTime int64   `json:"process_time_ms"`
}

// ModelInfo contains information about the loaded model
type ModelInfo struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Type      string    `json:"type"`
	Framework string    `json:"framework"`
	Features  int       `json:"feature_count"`
	Accuracy  float64   `json:"accuracy"`
	LoadedAt  time.Time `json:"loaded_at"`
	FilePath  string    `json:"file_path"`
	FileSize  int64     `json:"file_size"`
}

// FeatureExtractor extracts numerical features from emails for ML
type FeatureExtractor struct {
	spamWords     []string
	hamWords      []string
	urlRegex      *regexp.Regexp
	emailRegex    *regexp.Regexp
	phoneRegex    *regexp.Regexp
	currencyRegex *regexp.Regexp
}

// NewMLPlugin creates a new machine learning plugin
func NewMLPlugin() *MLPlugin {
	return &MLPlugin{
		enabled:  false,
		stats:    &MLStats{},
		features: NewFeatureExtractor(),
	}
}

// NewFeatureExtractor creates a new feature extractor
func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		spamWords: []string{
			"free", "urgent", "limited", "offer", "deal", "save", "money", "cash",
			"win", "winner", "congratulations", "prize", "guarantee", "risk-free",
			"viagra", "pharmacy", "pills", "weight", "loss", "diet", "loan",
			"credit", "debt", "investment", "million", "inheritance", "prince",
		},
		hamWords: []string{
			"meeting", "schedule", "conference", "project", "report", "update",
			"team", "colleague", "work", "office", "business", "professional",
			"regards", "sincerely", "thank", "please", "request", "information",
		},
		urlRegex:      regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`),
		emailRegex:    regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phoneRegex:    regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		currencyRegex: regexp.MustCompile(`[$£€¥]\d+|USD|EUR|GBP`),
	}
}

// Name returns the plugin name
func (ml *MLPlugin) Name() string {
	return "machine_learning"
}

// Version returns the plugin version
func (ml *MLPlugin) Version() string {
	return "1.0.0"
}

// Description returns plugin description
func (ml *MLPlugin) Description() string {
	return "Machine learning-based spam detection with support for multiple ML frameworks"
}

// Initialize sets up the plugin with configuration
func (ml *MLPlugin) Initialize(config *PluginConfig) error {
	ml.config = config
	ml.enabled = config.Enabled

	if !ml.enabled {
		return nil
	}

	if config.Settings == nil {
		return fmt.Errorf("ML plugin settings are required")
	}

	// Determine classifier type
	classifierType, ok := config.Settings["type"].(string)
	if !ok {
		classifierType = "simple" // Default to simple classifier
	}

	// Initialize appropriate classifier
	switch classifierType {
	case "simple":
		ml.backend = NewSimpleNaiveBayes()
	case "tensorflow":
		ml.backend = NewTensorFlowClassifier()
	case "pytorch":
		ml.backend = NewPyTorchClassifier()
	case "sklearn":
		ml.backend = NewSklearnClassifier()
	case "external":
		ml.backend = NewExternalMLService()
	default:
		return fmt.Errorf("unsupported classifier type: %s", classifierType)
	}

	// Initialize the classifier
	start := time.Now()
	err := ml.backend.Initialize(config.Settings)
	if err != nil {
		return fmt.Errorf("failed to initialize ML classifier: %v", err)
	}
	ml.stats.ModelLoadTime = time.Since(start).Milliseconds()
	ml.stats.LastModelUpdate = time.Now().Format(time.RFC3339)

	return nil
}

// IsHealthy checks if the plugin is ready
func (ml *MLPlugin) IsHealthy(ctx context.Context) error {
	if !ml.enabled {
		return fmt.Errorf("ML plugin not enabled")
	}

	if ml.backend == nil {
		return fmt.Errorf("ML classifier not initialized")
	}

	return ml.backend.IsHealthy()
}

// Cleanup releases resources
func (ml *MLPlugin) Cleanup() error {
	if ml.backend != nil {
		return ml.backend.Cleanup()
	}
	return nil
}

// Classify implements MLClassifier interface
func (ml *MLPlugin) Classify(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	result := &PluginResult{
		Name:        ml.Name(),
		Score:       0,
		Confidence:  0.7,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
		Rules:       []string{},
		ProcessTime: 0,
	}

	if !ml.enabled {
		result.Error = fmt.Errorf("plugin not enabled")
		result.ProcessTime = time.Since(start)
		return result, nil
	}

	// Extract features from email
	features := ml.features.ExtractFeatures(email)

	// Run ML prediction
	prediction, err := ml.backend.Predict(features)
	if err != nil {
		ml.stats.ClassificationsFailed++
		result.Error = err
		result.ProcessTime = time.Since(start)
		return result, err
	}

	ml.stats.ClassificationsTotal++

	// Convert prediction to score
	if prediction.IsSpam {
		result.Score = prediction.SpamScore * 25.0 // Scale to 0-25 range
		ml.stats.SpamDetected++
		result.Rules = append(result.Rules, fmt.Sprintf("ML classified as spam (confidence: %.2f)", prediction.Confidence))
	} else {
		result.Score = math.Max(0, (1.0-prediction.HamScore)*10.0) // Negative score for ham
		ml.stats.HamDetected++
	}

	result.Confidence = prediction.Confidence

	// Update running average confidence
	totalClassifications := float64(ml.stats.ClassificationsTotal)
	ml.stats.AverageConfidence = ((ml.stats.AverageConfidence * (totalClassifications - 1)) + prediction.Confidence) / totalClassifications

	// Add metadata
	result.Metadata["model_name"] = prediction.ModelName
	result.Metadata["feature_count"] = fmt.Sprintf("%d", prediction.Features)
	result.Metadata["spam_score"] = fmt.Sprintf("%.3f", prediction.SpamScore)
	result.Metadata["ham_score"] = fmt.Sprintf("%.3f", prediction.HamScore)

	// Add details
	result.Details["prediction"] = prediction
	result.Details["features_extracted"] = len(features)
	result.Details["is_spam"] = prediction.IsSpam

	result.ProcessTime = time.Since(start)
	return result, nil
}

// ExtractFeatures extracts numerical features from email for ML processing
func (fe *FeatureExtractor) ExtractFeatures(email *email.Email) []float64 {
	features := make([]float64, 0, 50) // Pre-allocate for ~50 features

	// Text content for analysis
	content := email.Subject + " " + email.Body
	contentLower := strings.ToLower(content)
	words := strings.Fields(contentLower)

	// Basic text statistics
	features = append(features, float64(len(email.Subject)))     // 0: Subject length
	features = append(features, float64(len(email.Body)))        // 1: Body length
	features = append(features, float64(len(words)))             // 2: Word count
	features = append(features, float64(len(email.Attachments))) // 3: Attachment count

	// Character ratios
	features = append(features, fe.calculateCapsRatio(content))        // 4: Caps ratio
	features = append(features, fe.calculateDigitRatio(content))       // 5: Digit ratio
	features = append(features, fe.calculatePunctuationRatio(content)) // 6: Punctuation ratio

	// Spam/Ham word frequencies
	spamWordCount := fe.countWordsInList(words, fe.spamWords)
	hamWordCount := fe.countWordsInList(words, fe.hamWords)
	features = append(features, float64(spamWordCount)) // 7: Spam words
	features = append(features, float64(hamWordCount))  // 8: Ham words

	// URL and contact analysis
	urls := fe.urlRegex.FindAllString(content, -1)
	emails := fe.emailRegex.FindAllString(content, -1)
	phones := fe.phoneRegex.FindAllString(content, -1)
	currencies := fe.currencyRegex.FindAllString(content, -1)

	features = append(features, float64(len(urls)))       // 9: URL count
	features = append(features, float64(len(emails)))     // 10: Email count
	features = append(features, float64(len(phones)))     // 11: Phone count
	features = append(features, float64(len(currencies))) // 12: Currency mentions

	// Special characters and patterns
	features = append(features, float64(strings.Count(content, "!"))) // 13: Exclamations
	features = append(features, float64(strings.Count(content, "?"))) // 14: Questions
	features = append(features, float64(strings.Count(content, "$"))) // 15: Dollar signs

	// HTML analysis
	htmlTags := strings.Count(strings.ToLower(content), "<")
	features = append(features, float64(htmlTags)) // 16: HTML tag count

	// Subject line analysis
	subjectWords := strings.Fields(strings.ToLower(email.Subject))
	subjectSpamWords := fe.countWordsInList(subjectWords, fe.spamWords)
	features = append(features, float64(subjectSpamWords)) // 17: Subject spam words

	// Header analysis
	suspiciousHeaders := 0
	if email.Headers["Reply-To"] != "" && email.Headers["Reply-To"] != email.From {
		suspiciousHeaders++
	}
	if strings.Contains(strings.ToLower(email.Headers["User-Agent"]), "outlook") {
		suspiciousHeaders++
	}
	features = append(features, float64(suspiciousHeaders)) // 18: Suspicious headers

	// Domain analysis
	domain := fe.extractDomain(email.From)
	features = append(features, fe.domainScore(domain)) // 19: Domain reputation score

	// Time-based features (if available)
	features = append(features, fe.timeScore(email.Headers["Date"])) // 20: Time score

	// Advanced text features
	features = append(features, fe.calculateReadabilityScore(content)) // 21: Readability
	features = append(features, fe.calculateLexicalDiversity(words))   // 22: Lexical diversity
	features = append(features, fe.calculateAverageWordLength(words))  // 23: Avg word length

	// Attachment analysis
	if len(email.Attachments) > 0 {
		features = append(features, fe.analyzeAttachments(email.Attachments)) // 24: Attachment risk
	} else {
		features = append(features, 0.0)
	}

	// Normalize features to 0-1 range for better ML performance
	return fe.normalizeFeatures(features)
}

// Helper methods for feature extraction
func (fe *FeatureExtractor) calculateCapsRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	caps := 0
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			caps++
		}
	}
	return float64(caps) / float64(len(text))
}

func (fe *FeatureExtractor) calculateDigitRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	digits := 0
	for _, r := range text {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return float64(digits) / float64(len(text))
}

func (fe *FeatureExtractor) calculatePunctuationRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	punct := strings.Count(text, "!") + strings.Count(text, "?") +
		strings.Count(text, ".") + strings.Count(text, ",") +
		strings.Count(text, ";") + strings.Count(text, ":")
	return float64(punct) / float64(len(text))
}

func (fe *FeatureExtractor) countWordsInList(words, wordList []string) int {
	count := 0
	wordMap := make(map[string]bool)
	for _, word := range wordList {
		wordMap[word] = true
	}

	for _, word := range words {
		if wordMap[word] {
			count++
		}
	}
	return count
}

func (fe *FeatureExtractor) extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return strings.ToLower(strings.Trim(parts[1], " <>"))
	}
	return ""
}

func (fe *FeatureExtractor) domainScore(domain string) float64 {
	// Simple domain reputation scoring
	suspiciousTlds := []string{".tk", ".ml", ".cf", ".ga", ".top", ".click", ".download"}
	for _, tld := range suspiciousTlds {
		if strings.HasSuffix(domain, tld) {
			return 0.8 // High suspicion
		}
	}

	reputableDomains := []string{"gmail.com", "outlook.com", "yahoo.com", "company.com"}
	for _, reputable := range reputableDomains {
		if strings.Contains(domain, reputable) {
			return 0.1 // Low suspicion
		}
	}

	return 0.3 // Neutral
}

func (fe *FeatureExtractor) timeScore(dateHeader string) float64 {
	// Placeholder for time-based analysis
	return 0.0
}

func (fe *FeatureExtractor) calculateReadabilityScore(text string) float64 {
	// Simplified readability score
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	sentences := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentences == 0 {
		sentences = 1
	}

	avgWordsPerSentence := float64(len(words)) / float64(sentences)
	return math.Min(avgWordsPerSentence/20.0, 1.0) // Normalize to 0-1
}

func (fe *FeatureExtractor) calculateLexicalDiversity(words []string) float64 {
	if len(words) == 0 {
		return 0
	}

	uniqueWords := make(map[string]bool)
	for _, word := range words {
		uniqueWords[word] = true
	}

	return float64(len(uniqueWords)) / float64(len(words))
}

func (fe *FeatureExtractor) calculateAverageWordLength(words []string) float64 {
	if len(words) == 0 {
		return 0
	}

	totalLength := 0
	for _, word := range words {
		totalLength += len(word)
	}

	avgLength := float64(totalLength) / float64(len(words))
	return math.Min(avgLength/10.0, 1.0) // Normalize to 0-1
}

func (fe *FeatureExtractor) analyzeAttachments(attachments []email.Attachment) float64 {
	risk := 0.0
	for _, attachment := range attachments {
		filename := strings.ToLower(attachment.Filename)

		// Executable files
		if strings.HasSuffix(filename, ".exe") || strings.HasSuffix(filename, ".scr") {
			risk += 0.8
		}

		// Archives
		if strings.HasSuffix(filename, ".zip") || strings.HasSuffix(filename, ".rar") {
			risk += 0.3
		}

		// Large files
		if attachment.Size > 1024*1024 { // 1MB
			risk += 0.2
		}
	}

	return math.Min(risk, 1.0)
}

func (fe *FeatureExtractor) normalizeFeatures(features []float64) []float64 {
	// Simple min-max normalization
	normalized := make([]float64, len(features))

	for i, feature := range features {
		// Most features are already in reasonable ranges
		// Apply log normalization for very large values
		if feature > 1000 {
			normalized[i] = math.Log(feature) / math.Log(1000)
		} else {
			normalized[i] = math.Min(feature/100.0, 1.0)
		}
	}

	return normalized
}

// GetStats returns plugin statistics
func (ml *MLPlugin) GetStats() *MLStats {
	return ml.stats
}

// Train implements MLClassifier interface (placeholder for future implementation)
func (ml *MLPlugin) Train(ctx context.Context, emails []email.Email, labels []bool) error {
	return fmt.Errorf("training not implemented yet")
}

// GetModelInfo returns information about the loaded model
func (ml *MLPlugin) GetModelInfo() (*ModelInfo, error) {
	if ml.backend == nil {
		return nil, fmt.Errorf("no classifier loaded")
	}
	return ml.backend.GetModelInfo()
}
