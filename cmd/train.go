package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/email"
	"github.com/zpo/spam-filter/pkg/learning"
)

var (
	trainSpamDir    string
	trainHamDir     string
	trainModelPath  string
	trainConfig     string
	trainReset      bool
	trainVerbose    bool
)

var trainCmd = &cobra.Command{
	Use:   "train",
	Short: "Train word frequency learning model",
	Long: `Train the Bayesian word frequency learning model using spam and ham email datasets.

The model learns word frequencies from training emails and can be used to improve spam detection accuracy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if trainSpamDir == "" && trainHamDir == "" {
			return fmt.Errorf("at least one of --spam-dir or --ham-dir must be specified")
		}

		// Load configuration
		cfg, err := config.LoadConfig(trainConfig)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Override model path if specified
		if trainModelPath != "" {
			cfg.Learning.ModelPath = trainModelPath
		}

		// Convert config to learning config
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

		// Create word frequency learner
		wf := learning.NewWordFrequency(learningConfig)

		// Load existing model if it exists and not resetting
		if !trainReset {
			if _, err := os.Stat(cfg.Learning.ModelPath); err == nil {
				if err := wf.LoadModel(cfg.Learning.ModelPath); err != nil {
					fmt.Printf("⚠️  Failed to load existing model: %v\n", err)
					fmt.Printf("🔄 Starting with fresh model...\n")
				} else {
					fmt.Printf("📚 Loaded existing model from: %s\n", cfg.Learning.ModelPath)
				}
			}
		}

		fmt.Printf("🧠 ZPO Word Frequency Training\n")
		fmt.Printf("═══════════════════════════════════════\n")
		if trainSpamDir != "" {
			fmt.Printf("📁 Spam directory: %s\n", trainSpamDir)
		}
		if trainHamDir != "" {
			fmt.Printf("📁 Ham directory: %s\n", trainHamDir)
		}
		fmt.Printf("💾 Model path: %s\n", cfg.Learning.ModelPath)
		if trainReset {
			fmt.Printf("🔄 Reset mode: Starting fresh\n")
		}
		fmt.Printf("\n")

		start := time.Now()
		var totalEmails int

		// Train on spam emails
		if trainSpamDir != "" {
			spamCount, err := trainDirectory(wf, trainSpamDir, true, trainVerbose)
			if err != nil {
				return fmt.Errorf("failed to train on spam emails: %v", err)
			}
			totalEmails += spamCount
			fmt.Printf("✅ Trained on %d spam emails\n", spamCount)
		}

		// Train on ham emails
		if trainHamDir != "" {
			hamCount, err := trainDirectory(wf, trainHamDir, false, trainVerbose)
			if err != nil {
				return fmt.Errorf("failed to train on ham emails: %v", err)
			}
			totalEmails += hamCount
			fmt.Printf("✅ Trained on %d ham emails\n", hamCount)
		}

		duration := time.Since(start)

		// Save the model
		if err := wf.SaveModel(cfg.Learning.ModelPath); err != nil {
			return fmt.Errorf("failed to save model: %v", err)
		}

		fmt.Printf("\n🎉 Training Complete!\n")
		fmt.Printf("📊 Total emails processed: %d\n", totalEmails)
		fmt.Printf("⏱️  Time taken: %v\n", duration)
		fmt.Printf("📈 Rate: %.0f emails/second\n", float64(totalEmails)/duration.Seconds())
		fmt.Printf("💾 Model saved to: %s\n", cfg.Learning.ModelPath)

		// Print model statistics
		fmt.Printf("\n")
		wf.PrintStats(os.Stdout)

		return nil
	},
}

// trainDirectory trains on all emails in a directory
func trainDirectory(wf *learning.WordFrequency, dir string, isSpam bool, verbose bool) (int, error) {
	var count int
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			return nil
		}
		
		// Check if file looks like an email
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".eml" && ext != ".msg" && ext != ".email" && ext != "" {
			return nil
		}
		
		// Parse email
		parser := email.NewParser()
		parsedEmail, err := parser.ParseFromFile(path)
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to parse %s: %v\n", path, err)
			}
			return nil // Skip but continue
		}
		
		// Train on email
		if isSpam {
			err = wf.TrainSpam(parsedEmail.Subject, parsedEmail.Body)
		} else {
			err = wf.TrainHam(parsedEmail.Subject, parsedEmail.Body)
		}
		
		if err != nil {
			if verbose {
				fmt.Printf("⚠️  Failed to train on %s: %v\n", path, err)
			}
			return nil // Skip but continue
		}
		
		count++
		if verbose && count%100 == 0 {
			fmt.Printf("📚 Processed %d emails...\n", count)
		}
		
		return nil
	})
	
	return count, err
}

func init() {
	trainCmd.Flags().StringVarP(&trainSpamDir, "spam-dir", "s", "", "Directory containing spam emails")
	trainCmd.Flags().StringVar(&trainHamDir, "ham-dir", "", "Directory containing ham emails")
	trainCmd.Flags().StringVarP(&trainModelPath, "model", "m", "", "Path to save/load model (overrides config)")
	trainCmd.Flags().StringVarP(&trainConfig, "config", "c", "", "Configuration file path")
	trainCmd.Flags().BoolVarP(&trainReset, "reset", "r", false, "Reset existing model and start fresh")
	trainCmd.Flags().BoolVarP(&trainVerbose, "verbose", "v", false, "Verbose output")
} 