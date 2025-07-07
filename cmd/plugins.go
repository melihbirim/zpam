package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/email"
	"github.com/zpo/spam-filter/pkg/filter"
	"github.com/zpo/spam-filter/pkg/plugins"
)

var (
	pluginConfigFile string
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage and test ZPO plugins",
	Long: `Commands to manage and test ZPO spam detection plugins.

ZPO supports various plugins for extending spam detection capabilities:
- SpamAssassin integration
- Rspamd integration  
- Custom rules engine
- VirusTotal reputation checking
- Machine learning models

Examples:
  zpo plugins list                    # List all available plugins
  zpo plugins test email.eml          # Test all enabled plugins on an email
  zpo plugins test-one spamassassin email.eml  # Test specific plugin
  zpo plugins stats                   # Show plugin execution statistics
  zpo plugins enable spamassassin     # Enable a plugin
  zpo plugins disable rspamd          # Disable a plugin`,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  "Display all registered plugins with their status and configuration.",
	Run:   runPluginsList,
}

var pluginsTestCmd = &cobra.Command{
	Use:   "test <email-file>",
	Short: "Test all enabled plugins on an email",
	Long: `Test all enabled plugins on a single email file and show detailed results.

This command will:
1. Load the specified email file
2. Execute all enabled plugins
3. Display individual plugin scores and results
4. Show the combined score and final determination`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsTest,
}

var pluginsTestOneCmd = &cobra.Command{
	Use:   "test-one <plugin-name> <email-file>",
	Short: "Test a specific plugin on an email",
	Long: `Test a single plugin on an email file and show detailed results.

Available plugins:
- spamassassin  : SpamAssassin integration
- rspamd        : Rspamd integration
- custom_rules  : Custom rules engine`,
	Args: cobra.ExactArgs(2),
	Run:  runPluginsTestOne,
}

var pluginsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show plugin execution statistics",
	Long:  "Display statistics for all plugins including execution count, timing, and error rates.",
	Run:   runPluginsStats,
}

var pluginsEnableCmd = &cobra.Command{
	Use:   "enable <plugin-name>",
	Short: "Enable a plugin",
	Long: `Enable a plugin in the configuration.

This command modifies the configuration to enable the specified plugin.
You may need to restart ZPO for changes to take effect.`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsEnable,
}

var pluginsDisableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Long: `Disable a plugin in the configuration.

This command modifies the configuration to disable the specified plugin.
You may need to restart ZPO for changes to take effect.`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsDisable,
}

func init() {
	// Add flags
	pluginsCmd.PersistentFlags().StringVarP(&pluginConfigFile, "config", "c", "", "Configuration file path")

	// Add subcommands
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsTestCmd)
	pluginsCmd.AddCommand(pluginsTestOneCmd)
	pluginsCmd.AddCommand(pluginsStatsCmd)
	pluginsCmd.AddCommand(pluginsEnableCmd)
	pluginsCmd.AddCommand(pluginsDisableCmd)

	// Add to root command
	rootCmd.AddCommand(pluginsCmd)
}

func runPluginsList(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create spam filter to initialize plugin manager
	sf := filter.NewSpamFilterWithConfig(cfg)

	fmt.Println("ZPO Plugins Status")
	fmt.Println("==================")
	fmt.Println()

	// Plugin system status
	if cfg.Plugins.Enabled {
		fmt.Printf("Plugin System: ENABLED\n")
		fmt.Printf("Score Method: %s\n", cfg.Plugins.ScoreMethod)
		fmt.Printf("Timeout: %dms\n", cfg.Plugins.Timeout)
		fmt.Printf("Max Concurrent: %d\n", cfg.Plugins.MaxConcurrent)
	} else {
		fmt.Printf("Plugin System: DISABLED\n")
	}
	fmt.Println()

	// Available plugins
	plugins := []struct {
		Name        string
		Enabled     bool
		Weight      float64
		Priority    int
		Timeout     int
		Description string
	}{
		{
			Name:        "spamassassin",
			Enabled:     cfg.Plugins.SpamAssassin.Enabled,
			Weight:      cfg.Plugins.SpamAssassin.Weight,
			Priority:    cfg.Plugins.SpamAssassin.Priority,
			Timeout:     cfg.Plugins.SpamAssassin.Timeout,
			Description: "SpamAssassin integration with spamc/spamassassin",
		},
		{
			Name:        "rspamd",
			Enabled:     cfg.Plugins.Rspamd.Enabled,
			Weight:      cfg.Plugins.Rspamd.Weight,
			Priority:    cfg.Plugins.Rspamd.Priority,
			Timeout:     cfg.Plugins.Rspamd.Timeout,
			Description: "Rspamd integration via HTTP API",
		},
		{
			Name:        "custom_rules",
			Enabled:     cfg.Plugins.CustomRules.Enabled,
			Weight:      cfg.Plugins.CustomRules.Weight,
			Priority:    cfg.Plugins.CustomRules.Priority,
			Timeout:     cfg.Plugins.CustomRules.Timeout,
			Description: "User-defined custom rules engine",
		},
		{
			Name:        "virustotal",
			Enabled:     cfg.Plugins.VirusTotal.Enabled,
			Weight:      cfg.Plugins.VirusTotal.Weight,
			Priority:    cfg.Plugins.VirusTotal.Priority,
			Timeout:     cfg.Plugins.VirusTotal.Timeout,
			Description: "VirusTotal URL/attachment reputation (not implemented)",
		},
		{
			Name:        "machine_learning",
			Enabled:     cfg.Plugins.MachineLearning.Enabled,
			Weight:      cfg.Plugins.MachineLearning.Weight,
			Priority:    cfg.Plugins.MachineLearning.Priority,
			Timeout:     cfg.Plugins.MachineLearning.Timeout,
			Description: "Machine learning model integration (not implemented)",
		},
	}

	fmt.Printf("%-20s %-8s %-8s %-8s %-8s %s\n", "PLUGIN", "ENABLED", "WEIGHT", "PRIORITY", "TIMEOUT", "DESCRIPTION")
	fmt.Println("-------------------------------------------------------------------------------")

	for _, plugin := range plugins {
		status := "NO"
		if plugin.Enabled {
			status = "YES"
		}

		fmt.Printf("%-20s %-8s %-8.1f %-8d %-8d %s\n",
			plugin.Name, status, plugin.Weight, plugin.Priority, plugin.Timeout, plugin.Description)
	}

	fmt.Println()
	fmt.Println("To enable plugins:")
	fmt.Println("  zpo plugins enable spamassassin")
	fmt.Println("  zpo plugins enable rspamd")
	fmt.Println()
	fmt.Println("To test plugins:")
	fmt.Println("  zpo plugins test examples/test_headers.eml")

	_ = sf // Use sf to avoid unused variable warning
}

func runPluginsTest(cmd *cobra.Command, args []string) {
	emailFile := args[0]

	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if !cfg.Plugins.Enabled {
		fmt.Println("Plugin system is disabled. Enable it in config.yaml first:")
		fmt.Println("plugins:")
		fmt.Println("  enabled: true")
		os.Exit(1)
	}

	// Create spam filter
	sf := filter.NewSpamFilterWithConfig(cfg)

	// Parse email
	parser := email.NewParser()
	emailObj, err := parser.ParseFromFile(emailFile)
	if err != nil {
		fmt.Printf("Error parsing email: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing plugins on: %s\n", emailFile)
	fmt.Printf("From: %s\n", emailObj.From)
	fmt.Printf("Subject: %s\n", emailObj.Subject)
	fmt.Println()

	// Test ZPO native scoring first
	start := time.Now()
	zpoScore := sf.CalculateSpamScore(emailObj)
	zpoTime := time.Since(start)
	zpoNormalized := sf.NormalizeScore(zpoScore)

	fmt.Printf("ZPO Native Score: %.2f (Level %d) in %v\n", zpoScore, zpoNormalized, zpoTime)
	fmt.Println()

	// Test plugins
	fmt.Println("Plugin Results:")
	fmt.Println("===============")

	// Create plugin manager and test
	pluginManager := plugins.NewPluginManager()

	// Register plugins
	pluginManager.RegisterPlugin(plugins.NewSpamAssassinPlugin())
	pluginManager.RegisterPlugin(plugins.NewRspamdPlugin())
	pluginManager.RegisterPlugin(plugins.NewCustomRulesPlugin())
	pluginManager.RegisterPlugin(plugins.NewVirusTotalPlugin())
	pluginManager.RegisterPlugin(plugins.NewMLPlugin())

	// Load configurations
	pluginConfigs := map[string]*plugins.PluginConfig{}
	if cfg.Plugins.SpamAssassin.Enabled {
		pluginConfigs["spamassassin"] = convertConfigToPluginConfig(cfg.Plugins.SpamAssassin)
	}
	if cfg.Plugins.Rspamd.Enabled {
		pluginConfigs["rspamd"] = convertConfigToPluginConfig(cfg.Plugins.Rspamd)
	}
	if cfg.Plugins.CustomRules.Enabled {
		pluginConfigs["custom_rules"] = convertConfigToPluginConfig(cfg.Plugins.CustomRules)
	}
	if cfg.Plugins.VirusTotal.Enabled {
		pluginConfigs["virustotal"] = convertConfigToPluginConfig(cfg.Plugins.VirusTotal)
	}
	if cfg.Plugins.MachineLearning.Enabled {
		pluginConfigs["machine_learning"] = convertConfigToPluginConfig(cfg.Plugins.MachineLearning)
	}

	if len(pluginConfigs) == 0 {
		fmt.Println("No plugins enabled. Enable plugins with:")
		fmt.Println("  zpo plugins enable spamassassin")
		return
	}

	err = pluginManager.LoadPlugins(pluginConfigs)
	if err != nil {
		fmt.Printf("Error loading plugins: %v\n", err)
		os.Exit(1)
	}

	// Execute plugins
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Plugins.Timeout)*time.Millisecond)
	defer cancel()

	start = time.Now()
	results, err := pluginManager.ExecuteAll(ctx, emailObj)
	totalTime := time.Since(start)

	if err != nil {
		fmt.Printf("Error executing plugins: %v\n", err)
		os.Exit(1)
	}

	// Display results
	var totalPluginScore float64
	for _, result := range results {
		status := "✓"
		if result.Error != nil {
			status = "✗"
		}

		fmt.Printf("%s %-15s Score: %6.2f  Confidence: %.2f  Time: %8v",
			status, result.Name, result.Score, result.Confidence, result.ProcessTime)

		if result.Error != nil {
			fmt.Printf("  Error: %v", result.Error)
		}
		fmt.Println()

		if len(result.Rules) > 0 {
			fmt.Printf("   Rules: %v\n", result.Rules)
		}

		if result.Error == nil {
			totalPluginScore += result.Score
		}
	}

	fmt.Println()

	// Combined score
	combinedScore, err := pluginManager.CombineScores(results)
	if err != nil {
		fmt.Printf("Error combining scores: %v\n", err)
	} else {
		fmt.Printf("Combined Plugin Score: %.2f\n", combinedScore)
	}

	finalScore := zpoScore + combinedScore
	finalNormalized := sf.NormalizeScore(finalScore)

	fmt.Printf("Final Combined Score: %.2f (Level %d)\n", finalScore, finalNormalized)
	fmt.Printf("Total Execution Time: %v\n", totalTime)

	// Recommendation
	fmt.Println()
	if finalNormalized >= 4 {
		fmt.Printf("🚨 RECOMMENDATION: SPAM (Score %d/5)\n", finalNormalized)
	} else {
		fmt.Printf("✅ RECOMMENDATION: HAM (Score %d/5)\n", finalNormalized)
	}
}

func runPluginsTestOne(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	emailFile := args[1]

	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Parse email
	parser := email.NewParser()
	emailObj, err := parser.ParseFromFile(emailFile)
	if err != nil {
		fmt.Printf("Error parsing email: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Testing plugin '%s' on: %s\n", pluginName, emailFile)
	fmt.Printf("From: %s\n", emailObj.From)
	fmt.Printf("Subject: %s\n", emailObj.Subject)
	fmt.Println()

	// Create and configure the specific plugin
	var plugin plugins.Plugin
	var pluginConfig *plugins.PluginConfig

	switch pluginName {
	case "spamassassin":
		plugin = plugins.NewSpamAssassinPlugin()
		pluginConfig = convertConfigToPluginConfig(cfg.Plugins.SpamAssassin)
	case "rspamd":
		plugin = plugins.NewRspamdPlugin()
		pluginConfig = convertConfigToPluginConfig(cfg.Plugins.Rspamd)
	case "custom_rules":
		plugin = plugins.NewCustomRulesPlugin()
		pluginConfig = convertConfigToPluginConfig(cfg.Plugins.CustomRules)
	case "virustotal":
		plugin = plugins.NewVirusTotalPlugin()
		pluginConfig = convertConfigToPluginConfig(cfg.Plugins.VirusTotal)
	case "machine_learning":
		plugin = plugins.NewMLPlugin()
		pluginConfig = convertConfigToPluginConfig(cfg.Plugins.MachineLearning)
	default:
		fmt.Printf("Unknown plugin: %s\n", pluginName)
		fmt.Println("Available plugins: spamassassin, rspamd, custom_rules, virustotal, machine_learning")
		os.Exit(1)
	}

	// Force enable for testing
	pluginConfig.Enabled = true

	// Initialize plugin
	err = plugin.Initialize(pluginConfig)
	if err != nil {
		fmt.Printf("Error initializing plugin: %v\n", err)
		os.Exit(1)
	}

	// Test plugin health
	ctx := context.Background()
	if err := plugin.IsHealthy(ctx); err != nil {
		fmt.Printf("Plugin health check failed: %v\n", err)
		fmt.Println("This plugin may not be properly configured or its dependencies may not be available.")
		fmt.Println()
	}

	// Execute plugin
	start := time.Now()

	// Determine plugin interface and execute accordingly
	var result *plugins.PluginResult
	switch p := plugin.(type) {
	case plugins.ContentAnalyzer:
		result, err = p.AnalyzeContent(ctx, emailObj)
	case plugins.ExternalEngine:
		result, err = p.Analyze(ctx, emailObj)
	case plugins.CustomRuleEngine:
		result, err = p.EvaluateRules(ctx, emailObj)
	case plugins.ReputationChecker:
		result, err = p.CheckReputation(ctx, emailObj)
	case plugins.MLClassifier:
		result, err = p.Classify(ctx, emailObj)
	default:
		fmt.Printf("Plugin %s does not implement a known interface\n", pluginName)
		os.Exit(1)
	}

	execTime := time.Since(start)

	if err != nil {
		fmt.Printf("Error executing plugin: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Plugin: %s v%s\n", plugin.Name(), plugin.Version())
	fmt.Printf("Description: %s\n", plugin.Description())
	fmt.Println()

	fmt.Printf("Score: %.2f\n", result.Score)
	fmt.Printf("Confidence: %.2f\n", result.Confidence)
	fmt.Printf("Execution Time: %v\n", execTime)

	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)
	}

	if len(result.Rules) > 0 {
		fmt.Printf("Triggered Rules: %v\n", result.Rules)
	}

	if len(result.Metadata) > 0 {
		fmt.Println("Metadata:")
		for key, value := range result.Metadata {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(result.Details) > 0 {
		fmt.Println("Details:")
		for key, value := range result.Details {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Cleanup
	plugin.Cleanup()
}

func runPluginsStats(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if !cfg.Plugins.Enabled {
		fmt.Println("Plugin system is disabled.")
		return
	}

	// Create spam filter to initialize plugin manager
	sf := filter.NewSpamFilterWithConfig(cfg)
	_ = sf // Placeholder until we implement stats tracking

	fmt.Println("Plugin Statistics")
	fmt.Println("=================")
	fmt.Println()
	fmt.Println("Note: Plugin statistics are currently tracked per-session.")
	fmt.Println("Run some plugin tests to see execution statistics here.")
	fmt.Println()
	fmt.Println("Commands to generate stats:")
	fmt.Println("  zpo plugins test examples/test_headers.eml")
	fmt.Println("  zpo plugins test-one spamassassin examples/test_headers.eml")
}

func runPluginsEnable(cmd *cobra.Command, args []string) {
	pluginName := args[0]

	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Enable the plugin
	var updated bool
	switch pluginName {
	case "spamassassin":
		cfg.Plugins.SpamAssassin.Enabled = true
		updated = true
	case "rspamd":
		cfg.Plugins.Rspamd.Enabled = true
		updated = true
	case "custom_rules":
		cfg.Plugins.CustomRules.Enabled = true
		updated = true
	case "virustotal":
		cfg.Plugins.VirusTotal.Enabled = true
		updated = true
	case "machine_learning":
		cfg.Plugins.MachineLearning.Enabled = true
		updated = true
	default:
		fmt.Printf("Unknown plugin: %s\n", pluginName)
		fmt.Println("Available plugins: spamassassin, rspamd, custom_rules, virustotal, machine_learning")
		os.Exit(1)
	}

	if updated {
		// Also enable the plugin system if it's not enabled
		if !cfg.Plugins.Enabled {
			cfg.Plugins.Enabled = true
			fmt.Println("Enabled plugin system")
		}

		// Save configuration
		if err := cfg.SaveConfig(pluginConfigFile); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Enabled plugin: %s\n", pluginName)
		fmt.Println("Configuration saved.")
	}
}

func runPluginsDisable(cmd *cobra.Command, args []string) {
	pluginName := args[0]

	// Load configuration
	cfg, err := filter.LoadConfigFromPath(pluginConfigFile)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Disable the plugin
	var updated bool
	switch pluginName {
	case "spamassassin":
		cfg.Plugins.SpamAssassin.Enabled = false
		updated = true
	case "rspamd":
		cfg.Plugins.Rspamd.Enabled = false
		updated = true
	case "custom_rules":
		cfg.Plugins.CustomRules.Enabled = false
		updated = true
	case "virustotal":
		cfg.Plugins.VirusTotal.Enabled = false
		updated = true
	case "machine_learning":
		cfg.Plugins.MachineLearning.Enabled = false
		updated = true
	default:
		fmt.Printf("Unknown plugin: %s\n", pluginName)
		fmt.Println("Available plugins: spamassassin, rspamd, custom_rules, virustotal, machine_learning")
		os.Exit(1)
	}

	if updated {
		// Save configuration
		if err := cfg.SaveConfig(pluginConfigFile); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Disabled plugin: %s\n", pluginName)
		fmt.Println("Configuration saved.")
	}
}

// Helper function to convert config to plugin config (copied from filter package)
func convertConfigToPluginConfig(cfg config.PluginConfig) *plugins.PluginConfig {
	return &plugins.PluginConfig{
		Enabled:  cfg.Enabled,
		Weight:   cfg.Weight,
		Priority: cfg.Priority,
		Timeout:  time.Duration(cfg.Timeout) * time.Millisecond,
		Settings: cfg.Settings,
	}
}
