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
	"github.com/zpo/spam-filter/pkg/filter"
)

var (
	trainSpamDir   string
	trainHamDir    string
	trainModelPath string
	trainConfig    string
	trainReset     bool
	trainVerbose   bool
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

		// Override model path if specified (only for file backend)
		if trainModelPath != "" && cfg.Learning.Backend == "file" {
			cfg.Learning.File.ModelPath = trainModelPath
		}

		// Create spam filter which will initialize the appropriate learner
		sf := filter.NewSpamFilterWithConfig(cfg)

		if sf == nil {
			return fmt.Errorf("failed to create spam filter")
		}

		// Reset model if requested
		if trainReset {
			if err := sf.ResetLearning(""); err != nil {
				fmt.Printf("⚠️  Failed to reset model: %v\n", err)
			} else {
				fmt.Printf("🔄 Reset model successfully\n")
			}
		}

		fmt.Printf("🧠 ZPO Bayesian Training (%s backend)\n", cfg.Learning.Backend)
		fmt.Printf("═══════════════════════════════════════\n")
		if trainSpamDir != "" {
			fmt.Printf("📁 Spam directory: %s\n", trainSpamDir)
		}
		if trainHamDir != "" {
			fmt.Printf("📁 Ham directory: %s\n", trainHamDir)
		}

		if cfg.Learning.Backend == "file" {
			fmt.Printf("💾 Model path: %s\n", cfg.Learning.File.ModelPath)
		} else {
			fmt.Printf("🔗 Redis URL: %s\n", cfg.Learning.Redis.RedisURL)
		}

		if trainReset {
			fmt.Printf("🔄 Reset mode: Starting fresh\n")
		}
		fmt.Printf("\n")

		start := time.Now()
		var totalEmails int

		// Train on spam emails
		if trainSpamDir != "" {
			spamCount, err := trainDirectory(sf, trainSpamDir, true, trainVerbose)
			if err != nil {
				return fmt.Errorf("failed to train on spam emails: %v", err)
			}
			totalEmails += spamCount
			fmt.Printf("✅ Trained on %d spam emails\n", spamCount)
		}

		// Train on ham emails
		if trainHamDir != "" {
			hamCount, err := trainDirectory(sf, trainHamDir, false, trainVerbose)
			if err != nil {
				return fmt.Errorf("failed to train on ham emails: %v", err)
			}
			totalEmails += hamCount
			fmt.Printf("✅ Trained on %d ham emails\n", hamCount)
		}

		duration := time.Since(start)

		// Save the model (for file backend)
		modelPath := ""
		if cfg.Learning.Backend == "file" {
			modelPath = cfg.Learning.File.ModelPath
			if err := sf.SaveModel(modelPath); err != nil {
				return fmt.Errorf("failed to save model: %v", err)
			}
		}

		fmt.Printf("\n🎉 Training Complete!\n")
		fmt.Printf("📊 Total emails processed: %d\n", totalEmails)
		fmt.Printf("⏱️  Time taken: %v\n", duration)
		fmt.Printf("📈 Rate: %.0f emails/second\n", float64(totalEmails)/duration.Seconds())

		if modelPath != "" {
			fmt.Printf("💾 Model saved to: %s\n", modelPath)
		}

		return nil
	},
}

// trainDirectory trains on all emails in a directory
func trainDirectory(sf *filter.SpamFilter, dir string, isSpam bool, verbose bool) (int, error) {
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
			err = sf.TrainSpam(parsedEmail.Subject, parsedEmail.Body, "")
		} else {
			err = sf.TrainHam(parsedEmail.Subject, parsedEmail.Body, "")
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
