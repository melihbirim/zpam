package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpam/spam-filter/pkg/filter"
)

var testConfigFile string

var testCmd = &cobra.Command{
	Use:   "test [email-file]",
	Short: "Test a single email for spam",
	Long:  `Test a single email file and get its spam score (1-5)`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		emailPath := args[0]

		// Load configuration
		cfg, err := filter.LoadConfigFromPath(testConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		spamFilter := filter.NewSpamFilterWithConfig(cfg)

		start := time.Now()
		score, err := spamFilter.TestEmail(emailPath)
		if err != nil {
			return fmt.Errorf("failed to test email: %v", err)
		}
		duration := time.Since(start)

		// Determine classification using config threshold
		threshold := cfg.Detection.SpamThreshold
		classification := "HAM (Clean)"
		if score >= threshold {
			classification = "SPAM"
		}

		fmt.Printf("ZPAM Test Results:\n")
		fmt.Printf("File: %s\n", emailPath)
		fmt.Printf("Score: %d/5\n", score)
		fmt.Printf("Classification: %s (threshold: %d)\n", classification, threshold)
		fmt.Printf("Processing time: %.2fms\n", float64(duration.Nanoseconds())/1e6)

		if testConfigFile != "" {
			fmt.Printf("Configuration: %s\n", testConfigFile)
		}

		return nil
	},
}

func init() {
	testCmd.Flags().StringVarP(&testConfigFile, "config", "c", "", "Configuration file path")
}
