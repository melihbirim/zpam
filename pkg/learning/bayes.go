package learning

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// WordFrequency tracks word frequencies for spam/ham classification
type WordFrequency struct {
	mu sync.RWMutex
	
	// Word counts
	spamWords map[string]int
	hamWords  map[string]int
	
	// Total counts
	totalSpamWords int
	totalHamWords  int
	totalSpamEmails int
	totalHamEmails  int
	
	// Configuration
	config *Config
	
	// Metadata
	lastTrained time.Time
	modelPath   string
}

// Config holds learning configuration
type Config struct {
	// Word processing
	MinWordLength    int     `json:"min_word_length"`
	MaxWordLength    int     `json:"max_word_length"`
	CaseSensitive    bool    `json:"case_sensitive"`
	
	// Learning parameters
	SpamThreshold    float64 `json:"spam_threshold"`
	MinWordCount     int     `json:"min_word_count"`
	SmoothingFactor  float64 `json:"smoothing_factor"`
	
	// Features
	UseSubjectWords  bool    `json:"use_subject_words"`
	UseBodyWords     bool    `json:"use_body_words"`
	UseHeaderWords   bool    `json:"use_header_words"`
	
	// Performance
	MaxVocabularySize int    `json:"max_vocabulary_size"`
}

// DefaultConfig returns default learning configuration
func DefaultConfig() *Config {
	return &Config{
		MinWordLength:    3,
		MaxWordLength:    20,
		CaseSensitive:    false,
		SpamThreshold:    0.7,
		MinWordCount:     2,
		SmoothingFactor:  1.0,
		UseSubjectWords:  true,
		UseBodyWords:     true,
		UseHeaderWords:   false,
		MaxVocabularySize: 10000,
	}
}

// NewWordFrequency creates a new word frequency learner
func NewWordFrequency(config *Config) *WordFrequency {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &WordFrequency{
		spamWords: make(map[string]int),
		hamWords:  make(map[string]int),
		config:    config,
	}
}

// WordStats contains statistics about a word
type WordStats struct {
	Word        string  `json:"word"`
	SpamCount   int     `json:"spam_count"`
	HamCount    int     `json:"ham_count"`
	SpamProb    float64 `json:"spam_prob"`
	HamProb     float64 `json:"ham_prob"`
	Score       float64 `json:"score"`
	Spamminess  float64 `json:"spamminess"`
}

// TrainSpam trains on a spam email
func (wf *WordFrequency) TrainSpam(subject, body string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	words := wf.extractWords(subject, body)
	
	for _, word := range words {
		wf.spamWords[word]++
		wf.totalSpamWords++
	}
	
	wf.totalSpamEmails++
	wf.lastTrained = time.Now()
	
	return nil
}

// TrainHam trains on a ham email
func (wf *WordFrequency) TrainHam(subject, body string) error {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	words := wf.extractWords(subject, body)
	
	for _, word := range words {
		wf.hamWords[word]++
		wf.totalHamWords++
	}
	
	wf.totalHamEmails++
	wf.lastTrained = time.Now()
	
	return nil
}

// ClassifyText returns spam probability for text
func (wf *WordFrequency) ClassifyText(subject, body string) float64 {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	if wf.totalSpamEmails == 0 && wf.totalHamEmails == 0 {
		return 0.5 // No training data
	}
	
	words := wf.extractWords(subject, body)
	if len(words) == 0 {
		return 0.5
	}
	
	// Calculate log probabilities to avoid underflow
	var logSpamProb, logHamProb float64
	
	// Prior probabilities
	spamPrior := float64(wf.totalSpamEmails) / float64(wf.totalSpamEmails+wf.totalHamEmails)
	hamPrior := float64(wf.totalHamEmails) / float64(wf.totalSpamEmails+wf.totalHamEmails)
	
	logSpamProb = math.Log(spamPrior)
	logHamProb = math.Log(hamPrior)
	
	// Word probabilities
	for _, word := range words {
		spamCount := wf.spamWords[word]
		hamCount := wf.hamWords[word]
		
		// Laplace smoothing
		spamWordProb := (float64(spamCount) + wf.config.SmoothingFactor) / 
						(float64(wf.totalSpamWords) + wf.config.SmoothingFactor*float64(len(wf.spamWords)))
		hamWordProb := (float64(hamCount) + wf.config.SmoothingFactor) / 
					   (float64(wf.totalHamWords) + wf.config.SmoothingFactor*float64(len(wf.hamWords)))
		
		logSpamProb += math.Log(spamWordProb)
		logHamProb += math.Log(hamWordProb)
	}
	
	// Convert back to probabilities
	spamProb := math.Exp(logSpamProb)
	hamProb := math.Exp(logHamProb)
	
	// Normalize
	totalProb := spamProb + hamProb
	if totalProb == 0 {
		return 0.5
	}
	
	return spamProb / totalProb
}

// extractWords extracts and normalizes words from text
func (wf *WordFrequency) extractWords(subject, body string) []string {
	var text string
	
	if wf.config.UseSubjectWords {
		text += subject + " "
	}
	if wf.config.UseBodyWords {
		text += body
	}
	
	if !wf.config.CaseSensitive {
		text = strings.ToLower(text)
	}
	
	// Extract words using regex
	wordRegex := regexp.MustCompile(`\b[a-zA-Z]{` + 
		fmt.Sprintf("%d,%d", wf.config.MinWordLength, wf.config.MaxWordLength) + `}\b`)
	
	matches := wordRegex.FindAllString(text, -1)
	
	// Deduplicate while preserving order
	seen := make(map[string]bool)
	var words []string
	
	for _, word := range matches {
		if !seen[word] {
			seen[word] = true
			words = append(words, word)
		}
	}
	
	return words
}

// GetWordStats returns statistics for a word
func (wf *WordFrequency) GetWordStats(word string) *WordStats {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	if !wf.config.CaseSensitive {
		word = strings.ToLower(word)
	}
	
	spamCount := wf.spamWords[word]
	hamCount := wf.hamWords[word]
	
	if spamCount == 0 && hamCount == 0 {
		return nil
	}
	
	// Calculate probabilities
	spamProb := float64(spamCount) / float64(wf.totalSpamWords)
	hamProb := float64(hamCount) / float64(wf.totalHamWords)
	
	// Spamminess score (0 = ham, 1 = spam)
	var spamminess float64
	if spamProb+hamProb > 0 {
		spamminess = spamProb / (spamProb + hamProb)
	}
	
	return &WordStats{
		Word:       word,
		SpamCount:  spamCount,
		HamCount:   hamCount,
		SpamProb:   spamProb,
		HamProb:    hamProb,
		Spamminess: spamminess,
	}
}

// GetTopSpamWords returns the most spammy words
func (wf *WordFrequency) GetTopSpamWords(limit int) []*WordStats {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	var words []*WordStats
	
	for word := range wf.spamWords {
		if stats := wf.GetWordStats(word); stats != nil {
			if stats.SpamCount >= wf.config.MinWordCount {
				words = append(words, stats)
			}
		}
	}
	
	// Sort by spamminess descending
	sort.Slice(words, func(i, j int) bool {
		return words[i].Spamminess > words[j].Spamminess
	})
	
	if limit > 0 && len(words) > limit {
		words = words[:limit]
	}
	
	return words
}

// GetTopHamWords returns the most ham words
func (wf *WordFrequency) GetTopHamWords(limit int) []*WordStats {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	var words []*WordStats
	
	for word := range wf.hamWords {
		if stats := wf.GetWordStats(word); stats != nil {
			if stats.HamCount >= wf.config.MinWordCount {
				words = append(words, stats)
			}
		}
	}
	
	// Sort by spamminess ascending (most ham-like)
	sort.Slice(words, func(i, j int) bool {
		return words[i].Spamminess < words[j].Spamminess
	})
	
	if limit > 0 && len(words) > limit {
		words = words[:limit]
	}
	
	return words
}

// ModelInfo contains model information
type ModelInfo struct {
	TotalSpamWords  int       `json:"total_spam_words"`
	TotalHamWords   int       `json:"total_ham_words"`
	TotalSpamEmails int       `json:"total_spam_emails"`
	TotalHamEmails  int       `json:"total_ham_emails"`
	VocabularySize  int       `json:"vocabulary_size"`
	LastTrained     time.Time `json:"last_trained"`
	Config          *Config   `json:"config"`
}

// GetModelInfo returns information about the trained model
func (wf *WordFrequency) GetModelInfo() *ModelInfo {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	// Count unique words
	vocabSize := len(wf.spamWords)
	for word := range wf.hamWords {
		if _, exists := wf.spamWords[word]; !exists {
			vocabSize++
		}
	}
	
	return &ModelInfo{
		TotalSpamWords:  wf.totalSpamWords,
		TotalHamWords:   wf.totalHamWords,
		TotalSpamEmails: wf.totalSpamEmails,
		TotalHamEmails:  wf.totalHamEmails,
		VocabularySize:  vocabSize,
		LastTrained:     wf.lastTrained,
		Config:          wf.config,
	}
}

// SaveModel saves the model to a file
func (wf *WordFrequency) SaveModel(path string) error {
	wf.mu.RLock()
	defer wf.mu.RUnlock()
	
	model := struct {
		SpamWords       map[string]int `json:"spam_words"`
		HamWords        map[string]int `json:"ham_words"`
		TotalSpamWords  int            `json:"total_spam_words"`
		TotalHamWords   int            `json:"total_ham_words"`
		TotalSpamEmails int            `json:"total_spam_emails"`
		TotalHamEmails  int            `json:"total_ham_emails"`
		LastTrained     time.Time      `json:"last_trained"`
		Config          *Config        `json:"config"`
	}{
		SpamWords:       wf.spamWords,
		HamWords:        wf.hamWords,
		TotalSpamWords:  wf.totalSpamWords,
		TotalHamWords:   wf.totalHamWords,
		TotalSpamEmails: wf.totalSpamEmails,
		TotalHamEmails:  wf.totalHamEmails,
		LastTrained:     wf.lastTrained,
		Config:          wf.config,
	}
	
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create model file: %v", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(model); err != nil {
		return fmt.Errorf("failed to encode model: %v", err)
	}
	
	wf.modelPath = path
	return nil
}

// LoadModel loads a model from a file
func (wf *WordFrequency) LoadModel(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open model file: %v", err)
	}
	defer file.Close()
	
	var model struct {
		SpamWords       map[string]int `json:"spam_words"`
		HamWords        map[string]int `json:"ham_words"`
		TotalSpamWords  int            `json:"total_spam_words"`
		TotalHamWords   int            `json:"total_ham_words"`
		TotalSpamEmails int            `json:"total_spam_emails"`
		TotalHamEmails  int            `json:"total_ham_emails"`
		LastTrained     time.Time      `json:"last_trained"`
		Config          *Config        `json:"config"`
	}
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&model); err != nil {
		return fmt.Errorf("failed to decode model: %v", err)
	}
	
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	wf.spamWords = model.SpamWords
	wf.hamWords = model.HamWords
	wf.totalSpamWords = model.TotalSpamWords
	wf.totalHamWords = model.TotalHamWords
	wf.totalSpamEmails = model.TotalSpamEmails
	wf.totalHamEmails = model.TotalHamEmails
	wf.lastTrained = model.LastTrained
	wf.modelPath = path
	
	if model.Config != nil {
		wf.config = model.Config
	}
	
	return nil
}

// Reset clears all learned data
func (wf *WordFrequency) Reset() {
	wf.mu.Lock()
	defer wf.mu.Unlock()
	
	wf.spamWords = make(map[string]int)
	wf.hamWords = make(map[string]int)
	wf.totalSpamWords = 0
	wf.totalHamWords = 0
	wf.totalSpamEmails = 0
	wf.totalHamEmails = 0
	wf.lastTrained = time.Time{}
}

// PrintStats prints model statistics
func (wf *WordFrequency) PrintStats(w io.Writer) {
	info := wf.GetModelInfo()
	
	fmt.Fprintf(w, "🧠 Word Frequency Learning Model\n")
	fmt.Fprintf(w, "════════════════════════════════════════\n")
	fmt.Fprintf(w, "Training Data:\n")
	fmt.Fprintf(w, "  Spam emails: %d\n", info.TotalSpamEmails)
	fmt.Fprintf(w, "  Ham emails: %d\n", info.TotalHamEmails)
	fmt.Fprintf(w, "  Spam words: %d\n", info.TotalSpamWords)
	fmt.Fprintf(w, "  Ham words: %d\n", info.TotalHamWords)
	fmt.Fprintf(w, "  Vocabulary size: %d\n", info.VocabularySize)
	
	if !info.LastTrained.IsZero() {
		fmt.Fprintf(w, "  Last trained: %s\n", info.LastTrained.Format("2006-01-02 15:04:05"))
	}
	
	fmt.Fprintf(w, "\nConfiguration:\n")
	fmt.Fprintf(w, "  Min word length: %d\n", info.Config.MinWordLength)
	fmt.Fprintf(w, "  Max word length: %d\n", info.Config.MaxWordLength)
	fmt.Fprintf(w, "  Spam threshold: %.2f\n", info.Config.SpamThreshold)
	fmt.Fprintf(w, "  Min word count: %d\n", info.Config.MinWordCount)
	fmt.Fprintf(w, "  Smoothing factor: %.2f\n", info.Config.SmoothingFactor)
	
	// Show top spam words
	fmt.Fprintf(w, "\n📈 Top Spam Words:\n")
	spamWords := wf.GetTopSpamWords(10)
	for i, word := range spamWords {
		fmt.Fprintf(w, "  %2d. %-15s (%.3f spamminess, %d/%d)\n", 
			i+1, word.Word, word.Spamminess, word.SpamCount, word.HamCount)
	}
	
	// Show top ham words
	fmt.Fprintf(w, "\n📉 Top Ham Words:\n")
	hamWords := wf.GetTopHamWords(10)
	for i, word := range hamWords {
		fmt.Fprintf(w, "  %2d. %-15s (%.3f spamminess, %d/%d)\n", 
			i+1, word.Word, word.Spamminess, word.SpamCount, word.HamCount)
	}
	
	fmt.Fprintf(w, "\n")
} 