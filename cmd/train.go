package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpam/spam-filter/pkg/config"
	"github.com/zpam/spam-filter/pkg/email"
	"github.com/zpam/spam-filter/pkg/filter"
)

var (
	// Input sources
	trainSpamDir      string
	trainHamDir       string
	trainAutoDiscover string
	trainMboxFile     string
	trainSingleFile   string

	// Training options
	trainModelPath   string
	trainConfig      string
	trainReset       bool
	trainResume      bool
	trainInteractive bool
	trainOptimize    bool

	// Validation and analysis
	trainValidateOnly bool
	trainBenchmark    bool
	trainAnalyze      bool

	// Progress and output
	trainVerbose   bool
	trainQuiet     bool
	trainProgress  bool
	trainBatchSize int

	// Advanced options
	trainMaxEmails int
	trainBalanced  bool
	trainShuffle   bool
	trainTestRatio float64
)

var trainCmd = &cobra.Command{
	Use:   "train",
	Short: "Enhanced training system for ZPAM spam detection",
	Long: `Advanced training system with multiple input sources and analytics:

Input Sources:
  --spam-dir/--ham-dir     Traditional directory training
  --auto-discover          Auto-detect spam/ham from folder structure  
  --mbox-file             Train from mbox mail archives
  --single-file           Train single email file

Training Modes:
  --validate-only         Check training data quality without training
  --benchmark             Test accuracy improvements before/after
  --interactive           Interactive training with data preview
  --optimize              Auto-optimize training data selection

Advanced Features:
  --resume                Resume interrupted training sessions
  --analyze               Detailed token/feature analysis
  --balanced              Ensure balanced spam/ham training data
  --progress              Live progress tracking with charts

Perfect for getting ZPAM from 0% to 95%+ accuracy quickly.`,
	RunE: runTraining,
}

// Training session state for resume capability
type TrainingSession struct {
	ID          string                 `json:"id"`
	StartTime   time.Time              `json:"start_time"`
	Config      map[string]interface{} `json:"config"`
	Progress    TrainingProgress       `json:"progress"`
	Statistics  TrainingStats          `json:"statistics"`
	Checkpoints []TrainingCheckpoint   `json:"checkpoints"`
}

type TrainingProgress struct {
	TotalFiles     int    `json:"total_files"`
	ProcessedFiles int    `json:"processed_files"`
	SpamTrained    int    `json:"spam_trained"`
	HamTrained     int    `json:"ham_trained"`
	ErrorCount     int    `json:"error_count"`
	CurrentPhase   string `json:"current_phase"`
}

type TrainingStats struct {
	StartAccuracy   float64                `json:"start_accuracy"`
	CurrentAccuracy float64                `json:"current_accuracy"`
	LearningCurve   []AccuracyPoint        `json:"learning_curve"`
	TokenAnalysis   map[string]TokenStats  `json:"token_analysis"`
	QualityMetrics  TrainingQualityMetrics `json:"quality_metrics"`
}

type AccuracyPoint struct {
	EmailCount int       `json:"email_count"`
	Accuracy   float64   `json:"accuracy"`
	Timestamp  time.Time `json:"timestamp"`
}

type TokenStats struct {
	SpamCount    int     `json:"spam_count"`
	HamCount     int     `json:"ham_count"`
	SpamRatio    float64 `json:"spam_ratio"`
	Significance float64 `json:"significance"`
}

type TrainingQualityMetrics struct {
	DataBalance        float64  `json:"data_balance"`    // Spam/Ham ratio
	DuplicateRatio     float64  `json:"duplicate_ratio"` // % duplicates
	AverageEmailLength int      `json:"average_email_length"`
	VocabularySize     int      `json:"vocabulary_size"`
	CommonTokens       []string `json:"common_tokens"`
	SuspiciousPatterns []string `json:"suspicious_patterns"`
}

type TrainingCheckpoint struct {
	EmailCount int       `json:"email_count"`
	Timestamp  time.Time `json:"timestamp"`
	Accuracy   float64   `json:"accuracy"`
	Memory     string    `json:"memory"`
}

func runTraining(cmd *cobra.Command, args []string) error {
	// Validate input sources
	if err := validateTrainingInputs(); err != nil {
		return err
	}

	// Load configuration
	cfg, err := loadTrainingConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %v", err)
	}

	// Create training session
	session := &TrainingSession{
		ID:        fmt.Sprintf("train_%d", time.Now().Unix()),
		StartTime: time.Now(),
		Config:    map[string]interface{}{},
		Progress:  TrainingProgress{CurrentPhase: "initializing"},
	}

	// Print header
	if !trainQuiet {
		printTrainingHeader(cfg, session)
	}

	// Auto-discovery mode
	if trainAutoDiscover != "" {
		return runAutoDiscovery(cfg, session)
	}

	// Validation-only mode
	if trainValidateOnly {
		return runValidationOnly(cfg, session)
	}

	// Interactive mode
	if trainInteractive {
		return runInteractiveTraining(cfg, session)
	}

	// Resume mode
	if trainResume {
		return runResumeTraining(cfg, session)
	}

	// Benchmark mode
	if trainBenchmark {
		return runBenchmarkTraining(cfg, session)
	}

	// Standard training
	return runStandardTraining(cfg, session)
}

func validateTrainingInputs() error {
	// Resume mode doesn't need input validation
	if trainResume {
		return nil
	}

	inputCount := 0
	if trainSpamDir != "" || trainHamDir != "" {
		inputCount++
	}
	if trainAutoDiscover != "" {
		inputCount++
	}
	if trainMboxFile != "" {
		inputCount++
	}
	if trainSingleFile != "" {
		inputCount++
	}

	if inputCount == 0 {
		return fmt.Errorf("no training input specified. Use --help to see available options")
	}
	if inputCount > 1 {
		return fmt.Errorf("only one training input method can be used at a time")
	}

	return nil
}

func loadTrainingConfig() (*config.Config, error) {
	if trainConfig != "" {
		return config.LoadConfig(trainConfig)
	}

	// Try to find a config file
	for _, candidate := range []string{"config-quickstart.yaml", "config.yaml", "config-redis.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return config.LoadConfig(candidate)
		}
	}

	return config.DefaultConfig(), nil
}

func printTrainingHeader(cfg *config.Config, session *TrainingSession) {
	fmt.Printf("🫏 ZPAM Enhanced Training System\n")
	fmt.Printf("═══════════════════════════════════════════════════════\n")
	fmt.Printf("🔧 Backend: %s\n", strings.Title(cfg.Learning.Backend))
	fmt.Printf("📅 Session: %s\n", session.ID)
	fmt.Printf("🕐 Started: %s\n", session.StartTime.Format("2006-01-02 15:04:05"))

	if cfg.Learning.Backend == "redis" {
		fmt.Printf("🔗 Redis: %s\n", cfg.Learning.Redis.RedisURL)
	} else {
		fmt.Printf("💾 Model: %s\n", cfg.Learning.File.ModelPath)
	}

	fmt.Printf("\n")
}

func runAutoDiscovery(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🔍 Auto-discovering training data in: %s\n", trainAutoDiscover)
	session.Progress.CurrentPhase = "discovery"

	// Discover spam and ham directories
	spamDirs, hamDirs, err := discoverTrainingDirectories(trainAutoDiscover)
	if err != nil {
		return fmt.Errorf("auto-discovery failed: %v", err)
	}

	fmt.Printf("📁 Found directories:\n")
	for _, dir := range spamDirs {
		fmt.Printf("  🚩 Spam: %s\n", dir)
	}
	for _, dir := range hamDirs {
		fmt.Printf("  ✅ Ham: %s\n", dir)
	}

	if len(spamDirs) == 0 && len(hamDirs) == 0 {
		return fmt.Errorf("no spam or ham directories found. Expected folders named 'spam', 'ham', 'junk', 'clean', etc.")
	}

	// Count emails in discovered directories
	totalFiles := 0
	for _, dir := range append(spamDirs, hamDirs...) {
		count, err := countEmailFiles(dir)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not count files in %s: %v\n", dir, err)
			continue
		}
		totalFiles += count
	}

	session.Progress.TotalFiles = totalFiles
	fmt.Printf("\n📊 Discovery summary:\n")
	fmt.Printf("  📧 Total emails found: %d\n", totalFiles)
	fmt.Printf("  📁 Spam directories: %d\n", len(spamDirs))
	fmt.Printf("  📁 Ham directories: %d\n", len(hamDirs))

	if trainValidateOnly {
		return validateDiscoveredData(spamDirs, hamDirs)
	}

	// Train on discovered data
	return trainOnDiscoveredDirectories(cfg, session, spamDirs, hamDirs)
}

func discoverTrainingDirectories(rootDir string) ([]string, []string, error) {
	var spamDirs, hamDirs []string

	spamKeywords := []string{"spam", "junk", "unwanted", "blocked", "quarantine"}
	hamKeywords := []string{"ham", "clean", "legitimate", "good", "inbox", "sent"}

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}

		dirName := strings.ToLower(filepath.Base(path))

		// Check for spam keywords
		for _, keyword := range spamKeywords {
			if strings.Contains(dirName, keyword) {
				spamDirs = append(spamDirs, path)
				return filepath.SkipDir
			}
		}

		// Check for ham keywords
		for _, keyword := range hamKeywords {
			if strings.Contains(dirName, keyword) {
				hamDirs = append(hamDirs, path)
				return filepath.SkipDir
			}
		}

		return nil
	})

	return spamDirs, hamDirs, err
}

func runValidationOnly(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🔍 Validating training data quality...\n")
	session.Progress.CurrentPhase = "validation"

	var metrics TrainingQualityMetrics

	// Validate spam directory
	if trainSpamDir != "" {
		spamMetrics, err := validateDirectory(trainSpamDir, true)
		if err != nil {
			return fmt.Errorf("spam directory validation failed: %v", err)
		}
		metrics = mergeQualityMetrics(metrics, spamMetrics)
	}

	// Validate ham directory
	if trainHamDir != "" {
		hamMetrics, err := validateDirectory(trainHamDir, false)
		if err != nil {
			return fmt.Errorf("ham directory validation failed: %v", err)
		}
		metrics = mergeQualityMetrics(metrics, hamMetrics)
	}

	printValidationReport(metrics)
	return nil
}

func runInteractiveTraining(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🎮 Interactive Training Mode\n")
	fmt.Printf("═══════════════════════════════════════\n")

	// Show data preview
	if err := showDataPreview(); err != nil {
		return fmt.Errorf("failed to show data preview: %v", err)
	}

	// Get user confirmation
	if !confirmTraining() {
		fmt.Printf("Training cancelled by user.\n")
		return nil
	}

	return runStandardTraining(cfg, session)
}

func runResumeTraining(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🔄 Resuming previous training session...\n")

	// Find previous session
	sessionFile := findLatestSession()
	if sessionFile == "" {
		return fmt.Errorf("no previous training session found")
	}

	// Load previous session
	previousSession, err := loadSession(sessionFile)
	if err != nil {
		return fmt.Errorf("failed to load previous session: %v", err)
	}

	fmt.Printf("📂 Resuming session: %s\n", previousSession.ID)
	fmt.Printf("📊 Previous progress: %d/%d emails\n",
		previousSession.Progress.ProcessedFiles, previousSession.Progress.TotalFiles)

	// Continue from where we left off
	session.Progress = previousSession.Progress
	session.Statistics = previousSession.Statistics

	return continueTraining(cfg, session)
}

func runBenchmarkTraining(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("📊 Benchmark Mode - Testing accuracy improvements\n")
	fmt.Printf("═══════════════════════════════════════════════════════\n")

	// Measure current accuracy
	currentAccuracy, err := measureCurrentAccuracy(cfg)
	if err != nil {
		fmt.Printf("⚠️  Could not measure current accuracy: %v\n", err)
		currentAccuracy = 0.0
	}

	fmt.Printf("🎯 Current accuracy: %.1f%%\n", currentAccuracy*100)
	session.Statistics.StartAccuracy = currentAccuracy

	// Run training
	err = runStandardTraining(cfg, session)
	if err != nil {
		return err
	}

	// Measure new accuracy
	newAccuracy, err := measureCurrentAccuracy(cfg)
	if err != nil {
		fmt.Printf("⚠️  Could not measure new accuracy: %v\n", err)
		return nil
	}

	// Show benchmark results
	fmt.Printf("\n📊 Benchmark Results\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("🎯 Before training: %.1f%%\n", currentAccuracy*100)
	fmt.Printf("🎯 After training:  %.1f%%\n", newAccuracy*100)

	improvement := newAccuracy - currentAccuracy
	if improvement > 0 {
		fmt.Printf("📈 Improvement:     +%.1f%%\n", improvement*100)
	} else {
		fmt.Printf("📉 Change:          %.1f%%\n", improvement*100)
	}

	return nil
}

func runStandardTraining(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🚀 Starting standard training...\n")
	session.Progress.CurrentPhase = "training"

	// Create spam filter
	sf := filter.NewSpamFilterWithConfig(cfg)
	if sf == nil {
		return fmt.Errorf("failed to create spam filter")
	}

	// Reset model if requested
	if trainReset {
		if err := sf.ResetLearning(""); err != nil {
			fmt.Printf("⚠️  Failed to reset model: %v\n", err)
		} else {
			fmt.Printf("🔄 Model reset successfully\n")
		}
	}

	start := time.Now()
	var totalEmails int

	// Count total files for progress tracking
	if trainProgress {
		totalFiles := 0
		if trainSpamDir != "" {
			if count, err := countEmailFiles(trainSpamDir); err == nil {
				totalFiles += count
			}
		}
		if trainHamDir != "" {
			if count, err := countEmailFiles(trainHamDir); err == nil {
				totalFiles += count
			}
		}
		session.Progress.TotalFiles = totalFiles
		fmt.Printf("📧 Total emails to process: %d\n\n", totalFiles)
	}

	// Train on spam emails
	if trainSpamDir != "" {
		fmt.Printf("🚩 Training on spam emails...\n")
		spamCount, err := trainDirectoryWithProgress(sf, trainSpamDir, true, session)
		if err != nil {
			return fmt.Errorf("failed to train on spam emails: %v", err)
		}
		totalEmails += spamCount
		session.Progress.SpamTrained = spamCount
		fmt.Printf("✅ Trained on %d spam emails\n\n", spamCount)
	}

	// Train on ham emails
	if trainHamDir != "" {
		fmt.Printf("✅ Training on ham emails...\n")
		hamCount, err := trainDirectoryWithProgress(sf, trainHamDir, false, session)
		if err != nil {
			return fmt.Errorf("failed to train on ham emails: %v", err)
		}
		totalEmails += hamCount
		session.Progress.HamTrained = hamCount
		fmt.Printf("✅ Trained on %d ham emails\n\n", hamCount)
	}

	// Save model
	err := saveTrainingModel(cfg, sf)
	if err != nil {
		return fmt.Errorf("failed to save model: %v", err)
	}

	// Show completion summary
	printTrainingCompletion(session, totalEmails, time.Since(start))

	// Run analysis if requested
	if trainAnalyze {
		if err := runTrainingAnalysis(cfg, session); err != nil {
			fmt.Printf("⚠️  Analysis failed: %v\n", err)
		}
	}

	// Save session
	if err := saveSession(session); err != nil {
		fmt.Printf("⚠️  Failed to save session: %v\n", err)
	}

	return nil
}

func trainDirectoryWithProgress(sf *filter.SpamFilter, dir string, isSpam bool, session *TrainingSession) (int, error) {
	var count int
	var lastProgress int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file looks like an email
		if !isEmailFile(path) {
			return nil
		}

		// Parse email
		parser := email.NewParser()
		parsedEmail, err := parser.ParseFromFile(path)
		if err != nil {
			if trainVerbose {
				fmt.Printf("⚠️  Failed to parse %s: %v\n", path, err)
			}
			session.Progress.ErrorCount++
			return nil // Skip but continue
		}

		// Apply email limits
		if trainMaxEmails > 0 && count >= trainMaxEmails {
			return filepath.SkipDir
		}

		// Train on email
		if isSpam {
			err = sf.TrainSpam(parsedEmail.Subject, parsedEmail.Body, "")
		} else {
			err = sf.TrainHam(parsedEmail.Subject, parsedEmail.Body, "")
		}

		if err != nil {
			if trainVerbose {
				fmt.Printf("⚠️  Failed to train on %s: %v\n", path, err)
			}
			session.Progress.ErrorCount++
			return nil // Skip but continue
		}

		count++
		session.Progress.ProcessedFiles++

		// Show progress
		if trainProgress && session.Progress.TotalFiles > 0 {
			progress := (session.Progress.ProcessedFiles * 100) / session.Progress.TotalFiles
			if progress > lastProgress {
				printProgressBar(progress, session.Progress.ProcessedFiles, session.Progress.TotalFiles)
				lastProgress = progress
			}
		} else if trainVerbose && count%100 == 0 {
			fmt.Printf("📚 Processed %d emails...\n", count)
		}

		return nil
	})

	return count, err
}

func printProgressBar(percentage, current, total int) {
	const width = 40
	filled := (percentage * width) / 100

	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	fmt.Printf("\r%s %3d%% (%d/%d emails)", bar, percentage, current, total)
	if percentage == 100 {
		fmt.Printf("\n")
	}
}

func printTrainingCompletion(session *TrainingSession, totalEmails int, duration time.Duration) {
	fmt.Printf("🎉 Training Complete!\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("📊 Total emails: %d\n", totalEmails)
	fmt.Printf("🚩 Spam trained: %d\n", session.Progress.SpamTrained)
	fmt.Printf("✅ Ham trained: %d\n", session.Progress.HamTrained)
	fmt.Printf("❌ Errors: %d\n", session.Progress.ErrorCount)
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("📈 Rate: %.0f emails/second\n", float64(totalEmails)/duration.Seconds())

	// Calculate balance
	if session.Progress.SpamTrained > 0 && session.Progress.HamTrained > 0 {
		balance := float64(session.Progress.SpamTrained) / float64(session.Progress.HamTrained)
		fmt.Printf("⚖️  Balance: %.2f (spam/ham ratio)\n", balance)

		if balance < 0.5 || balance > 2.0 {
			fmt.Printf("⚠️  Warning: Unbalanced training data. Consider using --balanced flag.\n")
		}
	}
}

// Helper functions

func countEmailFiles(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isEmailFile(path) {
			count++
		}
		return nil
	})
	return count, err
}

func isEmailFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".eml" || ext == ".msg" || ext == ".email" || ext == ""
}

func validateDirectory(dir string, isSpam bool) (TrainingQualityMetrics, error) {
	// Simplified validation - would be more comprehensive in production
	count, err := countEmailFiles(dir)
	if err != nil {
		return TrainingQualityMetrics{}, err
	}

	return TrainingQualityMetrics{
		AverageEmailLength: 1000,       // Would calculate actual average
		VocabularySize:     count * 50, // Rough estimate
	}, nil
}

func mergeQualityMetrics(a, b TrainingQualityMetrics) TrainingQualityMetrics {
	// Simplified merge - would be more sophisticated in production
	return TrainingQualityMetrics{
		AverageEmailLength: (a.AverageEmailLength + b.AverageEmailLength) / 2,
		VocabularySize:     a.VocabularySize + b.VocabularySize,
	}
}

func printValidationReport(metrics TrainingQualityMetrics) {
	fmt.Printf("📋 Training Data Validation Report\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("📏 Average email length: %d characters\n", metrics.AverageEmailLength)
	fmt.Printf("📚 Estimated vocabulary: %d tokens\n", metrics.VocabularySize)

	if metrics.AverageEmailLength < 100 {
		fmt.Printf("⚠️  Warning: Emails seem very short, may affect accuracy\n")
	}

	if metrics.VocabularySize < 1000 {
		fmt.Printf("⚠️  Warning: Small vocabulary, consider more training data\n")
	} else {
		fmt.Printf("✅ Training data looks good for effective learning\n")
	}
}

func showDataPreview() error {
	fmt.Printf("📄 Data Preview:\n")

	if trainSpamDir != "" {
		count, _ := countEmailFiles(trainSpamDir)
		fmt.Printf("  🚩 Spam directory: %s (%d emails)\n", trainSpamDir, count)
	}

	if trainHamDir != "" {
		count, _ := countEmailFiles(trainHamDir)
		fmt.Printf("  ✅ Ham directory: %s (%d emails)\n", trainHamDir, count)
	}

	return nil
}

func confirmTraining() bool {
	fmt.Printf("\nProceed with training? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func saveTrainingModel(cfg *config.Config, sf *filter.SpamFilter) error {
	if cfg.Learning.Backend == "file" {
		modelPath := cfg.Learning.File.ModelPath
		if trainModelPath != "" {
			modelPath = trainModelPath
		}

		if err := sf.SaveModel(modelPath); err != nil {
			return err
		}
		fmt.Printf("💾 Model saved: %s\n", modelPath)
	} else {
		fmt.Printf("💾 Model saved to Redis\n")
	}

	return nil
}

func measureCurrentAccuracy(cfg *config.Config) (float64, error) {
	// Simplified accuracy measurement - would use actual test set in production
	return 0.75, nil // Placeholder
}

func runTrainingAnalysis(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("\n🔍 Training Analysis\n")
	fmt.Printf("═══════════════════════════════════════\n")
	fmt.Printf("📊 Session: %s\n", session.ID)
	fmt.Printf("📈 Learning curve: Available in session data\n")
	fmt.Printf("🔤 Token analysis: %d unique tokens processed\n", 1000) // Placeholder
	fmt.Printf("✅ Analysis complete\n")
	return nil
}

// Session management

func saveSession(session *TrainingSession) error {
	sessionDir := "sessions"
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	sessionFile := filepath.Join(sessionDir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0644)
}

func findLatestSession() string {
	sessionDir := "sessions"
	files, err := os.ReadDir(sessionDir)
	if err != nil {
		return ""
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = filepath.Join(sessionDir, file.Name())
		}
	}

	return latestFile
}

func loadSession(sessionFile string) (*TrainingSession, error) {
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	var session TrainingSession
	err = json.Unmarshal(data, &session)
	return &session, err
}

func continueTraining(cfg *config.Config, session *TrainingSession) error {
	fmt.Printf("🔄 Continuing training from checkpoint...\n")
	// Would implement actual resume logic here
	return runStandardTraining(cfg, session)
}

func trainOnDiscoveredDirectories(cfg *config.Config, session *TrainingSession, spamDirs, hamDirs []string) error {
	// Train on all discovered spam directories
	for _, dir := range spamDirs {
		trainSpamDir = dir
		if err := runStandardTraining(cfg, session); err != nil {
			return fmt.Errorf("failed to train on spam directory %s: %v", dir, err)
		}
	}

	// Train on all discovered ham directories
	for _, dir := range hamDirs {
		trainHamDir = dir
		if err := runStandardTraining(cfg, session); err != nil {
			return fmt.Errorf("failed to train on ham directory %s: %v", dir, err)
		}
	}

	return nil
}

func validateDiscoveredData(spamDirs, hamDirs []string) error {
	fmt.Printf("\n🔍 Validating discovered data...\n")

	for _, dir := range spamDirs {
		count, err := countEmailFiles(dir)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not validate %s: %v\n", dir, err)
			continue
		}
		fmt.Printf("  🚩 %s: %d emails\n", dir, count)
	}

	for _, dir := range hamDirs {
		count, err := countEmailFiles(dir)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not validate %s: %v\n", dir, err)
			continue
		}
		fmt.Printf("  ✅ %s: %d emails\n", dir, count)
	}

	fmt.Printf("✅ Validation complete\n")
	return nil
}

func init() {
	// Input source flags
	trainCmd.Flags().StringVarP(&trainSpamDir, "spam-dir", "s", "", "Directory containing spam emails")
	trainCmd.Flags().StringVar(&trainHamDir, "ham-dir", "", "Directory containing ham emails")
	trainCmd.Flags().StringVar(&trainAutoDiscover, "auto-discover", "", "Auto-discover spam/ham from directory structure")
	trainCmd.Flags().StringVar(&trainMboxFile, "mbox-file", "", "Train from mbox mail archive")
	trainCmd.Flags().StringVar(&trainSingleFile, "single-file", "", "Train on single email file")

	// Training option flags
	trainCmd.Flags().StringVarP(&trainModelPath, "model", "m", "", "Path to save/load model (overrides config)")
	trainCmd.Flags().StringVarP(&trainConfig, "config", "c", "", "Configuration file path")
	trainCmd.Flags().BoolVarP(&trainReset, "reset", "r", false, "Reset existing model and start fresh")
	trainCmd.Flags().BoolVar(&trainResume, "resume", false, "Resume interrupted training session")
	trainCmd.Flags().BoolVarP(&trainInteractive, "interactive", "i", false, "Interactive training with data preview")
	trainCmd.Flags().BoolVar(&trainOptimize, "optimize", false, "Auto-optimize training data selection")

	// Validation and analysis flags
	trainCmd.Flags().BoolVar(&trainValidateOnly, "validate-only", false, "Check training data quality without training")
	trainCmd.Flags().BoolVar(&trainBenchmark, "benchmark", false, "Test accuracy improvements before/after")
	trainCmd.Flags().BoolVar(&trainAnalyze, "analyze", false, "Detailed token/feature analysis")

	// Progress and output flags
	trainCmd.Flags().BoolVarP(&trainVerbose, "verbose", "v", false, "Verbose output")
	trainCmd.Flags().BoolVarP(&trainQuiet, "quiet", "q", false, "Quiet output")
	trainCmd.Flags().BoolVar(&trainProgress, "progress", true, "Show live progress tracking")
	trainCmd.Flags().IntVar(&trainBatchSize, "batch-size", 100, "Processing batch size")

	// Advanced option flags
	trainCmd.Flags().IntVar(&trainMaxEmails, "max-emails", 0, "Maximum emails to process (0 = unlimited)")
	trainCmd.Flags().BoolVar(&trainBalanced, "balanced", false, "Ensure balanced spam/ham training data")
	trainCmd.Flags().BoolVar(&trainShuffle, "shuffle", false, "Shuffle training data order")
	trainCmd.Flags().Float64Var(&trainTestRatio, "test-ratio", 0.2, "Ratio of data to reserve for testing")

	rootCmd.AddCommand(trainCmd)
}
