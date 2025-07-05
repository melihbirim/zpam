package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/filter"
)

var (
	inputPath  string
	outputPath string
	spamPath   string
	threshold  int
	configFile string
)

var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: "Filter emails for spam",
	Long:  `Process emails and filter spam based on ZPO's fast algorithm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if inputPath == "" {
			return fmt.Errorf("input path is required")
		}

		// Load configuration
		cfg, err := filter.LoadConfigFromPath(configFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Use config threshold if not overridden by command line
		if !cmd.Flags().Changed("threshold") {
			threshold = cfg.Detection.SpamThreshold
		}

		spamFilter := filter.NewSpamFilterWithConfig(cfg)
		
		// Process emails
		start := time.Now()
		results, err := spamFilter.ProcessEmails(inputPath, outputPath, spamPath, threshold)
		if err != nil {
			return fmt.Errorf("failed to process emails: %v", err)
		}
		duration := time.Since(start)

		// Print results
		fmt.Printf("ZPO Processing Complete!\n")
		fmt.Printf("Emails processed: %d\n", results.Total)
		fmt.Printf("Spam detected: %d\n", results.Spam)
		fmt.Printf("Ham (clean): %d\n", results.Ham)
		fmt.Printf("Average processing time: %.2fms per email\n", 
			float64(duration.Nanoseconds())/float64(results.Total)/1e6)
		fmt.Printf("Total time: %v\n", duration)
		
		if configFile != "" {
			fmt.Printf("Configuration: %s\n", configFile)
		}

		return nil
	},
}

func init() {
	filterCmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input directory or file path")
	filterCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory for clean emails")
	filterCmd.Flags().StringVarP(&spamPath, "spam", "s", "", "Spam directory for filtered emails")
	filterCmd.Flags().IntVarP(&threshold, "threshold", "t", 4, "Spam threshold (4-5 = spam)")
	filterCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	
	filterCmd.MarkFlagRequired("input")
} 