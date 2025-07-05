package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Generate and manage ZPO configuration files`,
}

var configGenCmd = &cobra.Command{
	Use:   "generate [config-file]",
	Short: "Generate default configuration file",
	Long:  `Generate a default configuration file with all options and documentation`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "config.yaml"
		if len(args) > 0 {
			configPath = args[0]
		}
		
		// Check if file already exists
		if _, err := os.Stat(configPath); err == nil {
			overwrite, _ := cmd.Flags().GetBool("force")
			if !overwrite {
				return fmt.Errorf("config file already exists: %s (use --force to overwrite)", configPath)
			}
		}
		
		// Generate default config
		defaultConfig := config.DefaultConfig()
		
		// Save to file
		err := defaultConfig.SaveConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}
		
		fmt.Printf("✅ Configuration file generated: %s\n", configPath)
		fmt.Printf("📝 Edit the file to customize spam detection rules\n")
		fmt.Printf("🚀 Use 'zpo filter --config %s' to use the configuration\n", configPath)
		
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	Long:  `Validate a configuration file for syntax and logical errors`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]
		
		// Load and validate config
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("❌ Configuration validation failed: %v", err)
		}
		
		// Additional validation checks
		warnings := validateConfigLogic(cfg)
		
		fmt.Printf("✅ Configuration is valid: %s\n", configPath)
		
		if len(warnings) > 0 {
			fmt.Printf("\n⚠️  Warnings:\n")
			for _, warning := range warnings {
				fmt.Printf("  - %s\n", warning)
			}
		}
		
		// Print summary
		fmt.Printf("\n📊 Configuration Summary:\n")
		fmt.Printf("  Spam threshold: %d\n", cfg.Detection.SpamThreshold)
		fmt.Printf("  High-risk keywords: %d\n", len(cfg.Detection.Keywords.HighRisk))
		fmt.Printf("  Trusted domains: %d\n", len(cfg.Lists.TrustedDomains))
		fmt.Printf("  Whitelist emails: %d\n", len(cfg.Lists.WhitelistEmails))
		fmt.Printf("  Blacklist emails: %d\n", len(cfg.Lists.BlacklistEmails))
		
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [config-file]",
	Short: "Show current configuration",
	Long:  `Display the current configuration with all values`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg *config.Config
		var err error
		
		if len(args) > 0 {
			cfg, err = config.LoadConfig(args[0])
			if err != nil {
				return fmt.Errorf("failed to load config: %v", err)
			}
			fmt.Printf("Configuration: %s\n\n", args[0])
		} else {
			cfg = config.DefaultConfig()
			fmt.Printf("Default Configuration:\n\n")
		}
		
		// Display key settings
		fmt.Printf("🎯 Spam Detection:\n")
		fmt.Printf("  Threshold: %d (4-5 = spam)\n", cfg.Detection.SpamThreshold)
		fmt.Printf("  Features enabled: %v\n", getEnabledFeatures(cfg))
		
		fmt.Printf("\n📊 Scoring Weights:\n")
		weights := cfg.Detection.Weights
		fmt.Printf("  Subject keywords: %.1f\n", weights.SubjectKeywords)
		fmt.Printf("  Body keywords: %.1f\n", weights.BodyKeywords)
		fmt.Printf("  Domain reputation: %.1f\n", weights.DomainReputation)
		fmt.Printf("  Frequency penalty: %.1f\n", weights.FrequencyPenalty)
		
		fmt.Printf("\n📋 Lists:\n")
		fmt.Printf("  Trusted domains: %d\n", len(cfg.Lists.TrustedDomains))
		fmt.Printf("  Whitelist emails: %d\n", len(cfg.Lists.WhitelistEmails))
		fmt.Printf("  Blacklist emails: %d\n", len(cfg.Lists.BlacklistEmails))
		
		fmt.Printf("\n⚡ Performance:\n")
		fmt.Printf("  Max concurrent: %d\n", cfg.Performance.MaxConcurrentEmails)
		fmt.Printf("  Timeout: %dms\n", cfg.Performance.TimeoutMs)
		fmt.Printf("  Cache size: %d\n", cfg.Performance.CacheSize)
		
		return nil
	},
}

// validateConfigLogic performs additional logical validation
func validateConfigLogic(cfg *config.Config) []string {
	var warnings []string
	
	// Check for potential issues
	if cfg.Detection.SpamThreshold == 5 {
		warnings = append(warnings, "Spam threshold is set to maximum (5) - might miss some spam")
	}
	
	if cfg.Detection.SpamThreshold == 1 {
		warnings = append(warnings, "Spam threshold is set to minimum (1) - might flag too much as spam")
	}
	
	if len(cfg.Detection.Keywords.HighRisk) == 0 {
		warnings = append(warnings, "No high-risk keywords defined")
	}
	
	if cfg.Performance.MaxConcurrentEmails > 50 {
		warnings = append(warnings, "High concurrency setting might impact performance")
	}
	
	if cfg.Performance.TimeoutMs < 1000 {
		warnings = append(warnings, "Low timeout setting might cause failures on slow systems")
	}
	
	// Check for conflicting lists
	for _, email := range cfg.Lists.WhitelistEmails {
		for _, blackEmail := range cfg.Lists.BlacklistEmails {
			if email == blackEmail {
				warnings = append(warnings, fmt.Sprintf("Email %s is in both whitelist and blacklist", email))
			}
		}
	}
	
	return warnings
}

// getEnabledFeatures returns a list of enabled features
func getEnabledFeatures(cfg *config.Config) []string {
	features := []string{}
	
	if cfg.Detection.Features.KeywordDetection {
		features = append(features, "keywords")
	}
	if cfg.Detection.Features.HeaderAnalysis {
		features = append(features, "headers")
	}
	if cfg.Detection.Features.AttachmentScan {
		features = append(features, "attachments")
	}
	if cfg.Detection.Features.DomainCheck {
		features = append(features, "domains")
	}
	if cfg.Detection.Features.FrequencyTracking {
		features = append(features, "frequency")
	}
	if cfg.Detection.Features.LearningMode {
		features = append(features, "learning")
	}
	
	return features
}

func init() {
	// Add subcommands
	configCmd.AddCommand(configGenCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	
	// Add flags
	configGenCmd.Flags().Bool("force", false, "Overwrite existing config file")
} 