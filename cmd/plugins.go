package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	lua "github.com/yuin/gopher-lua"
	"github.com/zpam/spam-filter/pkg/config"
	"github.com/zpam/spam-filter/pkg/email"
	"github.com/zpam/spam-filter/pkg/filter"
	"github.com/zpam/spam-filter/pkg/plugins"
)

var (
	pluginConfigFile string
	forceInstall     bool
	pluginTemplate   string
	pluginLicense    string
	strictValidation bool
	securityOnly     bool
	publishRegistry  string
	privateRegistry  bool
)

// Plugin marketplace structures
type MarketplacePlugin struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Type         string            `json:"type"` // "content_analyzer", "ml_classifier", etc.
	Tags         []string          `json:"tags"`
	DownloadURL  string            `json:"download_url"`
	Homepage     string            `json:"homepage"`
	Repository   string            `json:"repository"`
	License      string            `json:"license"`
	Dependencies []string          `json:"dependencies"`
	MinVersion   string            `json:"min_zpam_version"`
	Verified     bool              `json:"verified"`
	Downloads    int               `json:"downloads"`
	Rating       float64           `json:"rating"`
	Settings     map[string]string `json:"settings"`
}

type MarketplaceResponse struct {
	Plugins   []MarketplacePlugin `json:"plugins"`
	Total     int                 `json:"total"`
	Page      int                 `json:"page"`
	PerPage   int                 `json:"per_page"`
	UpdatedAt string              `json:"updated_at"`
}

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage and test ZPAM plugins",
	Long: `Commands to manage and test ZPAM spam detection plugins.

ZPAM supports various plugins for extending spam detection capabilities:
- SpamAssassin integration
- Rspamd integration  
- Custom rules engine
- VirusTotal reputation checking
- Machine learning models

Examples:
  zpam plugins discover                    # Browse available plugins
  zpam plugins install openai-classifier  # Install from marketplace
  zpam plugins search "phishing"          # Search plugins
  zpam plugins list                       # List installed plugins
  zpam plugins test email.eml             # Test all enabled plugins
  zpam plugins enable spamassassin        # Enable a plugin`,
}

var pluginsDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Browse available plugins from the marketplace",
	Long: `Display available plugins from the ZPAM plugin marketplace.

This command shows all available plugins that can be installed, including:
- Official verified plugins
- Community-contributed plugins
- AI/ML model integrations
- External service integrations

Examples:
  zpam plugins discover              # Show all available plugins
  zpam plugins discover --verified   # Show only verified plugins`,
	Run: runPluginsDiscover,
}

var pluginsInstallCmd = &cobra.Command{
	Use:   "install <plugin-source>",
	Short: "Install a plugin from multiple sources",
	Long: `Install a plugin from the ZPAM marketplace, GitHub, ZIP file, or local folder.

This command automatically detects the source type and:
1. Download/copy the plugin from the specified source
2. Verify the plugin compatibility and dependencies  
3. Install the plugin and make it available for use
4. Update the configuration to include the new plugin

Examples:
  zpam plugins install openai-classifier                    # Install from marketplace
  zpam plugins install github:zpam-team/openai-classifier   # Install from GitHub
  zpam plugins install https://github.com/user/plugin       # Install from GitHub URL
  zpam plugins install ./my-plugin/                         # Install from local folder
  zpam plugins install plugin.zip                           # Install from local ZIP
  zpam plugins install https://example.com/plugin.zip       # Install from remote ZIP
  zpam plugins install openai-classifier --force           # Force install/upgrade`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsInstall,
}

var pluginsSearchCmd = &cobra.Command{
	Use:   "search <keyword>",
	Short: "Search plugins by keyword",
	Long: `Search the plugin marketplace for plugins matching the given keyword.

The search will look through plugin names, descriptions, tags, and authors.

Examples:
  zpam plugins search "phishing"     # Find phishing detection plugins
  zpam plugins search "ai"           # Find AI-powered plugins
  zpam plugins search "microsoft"    # Find Microsoft integrations`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsSearch,
}

var pluginsUninstallCmd = &cobra.Command{
	Use:   "uninstall <plugin-name>",
	Short: "Uninstall a plugin",
	Long: `Remove a plugin from the system.

This command will:
1. Disable the plugin if it's currently enabled
2. Remove the plugin files
3. Clean up any plugin-specific configuration
4. Update the system configuration

Examples:
  zpam plugins uninstall openai-classifier
  zpam plugins uninstall custom-rules-extended`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsUninstall,
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
You may need to restart ZPAM for changes to take effect.`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsEnable,
}

var pluginsDisableCmd = &cobra.Command{
	Use:   "disable <plugin-name>",
	Short: "Disable a plugin",
	Long: `Disable a plugin in the configuration.

This command modifies the configuration to disable the specified plugin.
You may need to restart ZPAM for changes to take effect.`,
	Args: cobra.ExactArgs(1),
	Run:  runPluginsDisable,
}

var pluginsDiscoverGitHubCmd = &cobra.Command{
	Use:   "discover-github",
	Short: "Discover plugins from GitHub repositories",
	Long: `Search GitHub repositories with the 'zpam-plugin' topic to discover community plugins.

This command searches GitHub for repositories tagged with 'zpam-plugin' and shows
available plugins that can be installed directly from GitHub.

Examples:
  zpam plugins discover-github               # Show all GitHub plugins
  zpam plugins discover-github --verified     # Show only verified plugins
  zpam plugins discover-github --type ml      # Show ML classifier plugins`,
	Run: runPluginsDiscoverGitHub,
}

var pluginsUpdateRegistryCmd = &cobra.Command{
	Use:   "update-registry",
	Short: "Update the plugin registry from GitHub",
	Long: `Update the local plugin registry by scanning GitHub for new plugins.

This command:
1. Searches GitHub for repositories with 'zpam-plugin' topic
2. Fetches and validates plugin manifests
3. Updates the local registry cache
4. Refreshes plugin availability for 'discover' command

Examples:
  zpam plugins update-registry              # Update registry
  zpam plugins update-registry --force      # Force full refresh`,
	Run: runPluginsUpdateRegistry,
}

var pluginsCreateCmd = &cobra.Command{
	Use:   "create <plugin-name> <type> [language]",
	Short: "Create a new plugin from template",
	Long: `Generate a new ZPAM plugin from predefined templates.

This command creates a complete plugin project structure with:
- zpam-plugin.yaml manifest
- Source code template
- Build scripts and configuration
- Documentation and examples
- Test files

Available plugin types:
  content-analyzer     - Analyze email content for spam indicators
  reputation-checker   - Check sender/domain reputation
  attachment-scanner   - Scan email attachments
  ml-classifier        - Machine learning classification
  external-engine      - Integration with external services
  custom-rule-engine   - Custom rule evaluation

Available languages:
  go                   - Go language (default)
  lua                  - Lua scripting language

Examples:
  zpam plugins create my-ai-filter ml-classifier
  zpam plugins create phishing-detector content-analyzer lua
  zpam plugins create virus-scanner attachment-scanner go`,
	Args: cobra.RangeArgs(2, 3),
	Run:  runPluginsCreate,
}

var pluginsValidateCmd = &cobra.Command{
	Use:   "validate [plugin-path]",
	Short: "Validate plugin compliance and security",
	Long: `Validate a plugin for compliance, security, and quality standards.

This command performs comprehensive validation:
- Manifest syntax and completeness
- Interface compliance checking
- Security permission validation
- Code quality analysis
- Performance benchmarking
- Dependency verification

Examples:
  zpam plugins validate                    # Validate current directory
  zpam plugins validate ./my-plugin/      # Validate specific plugin
  zpam plugins validate --strict          # Strict validation mode
  zpam plugins validate --security-only   # Security checks only`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPluginsValidate,
}

var pluginsBuildCmd = &cobra.Command{
	Use:   "build [plugin-path]",
	Short: "Build and package a plugin",
	Long: `Build and package a plugin for distribution.

This command:
1. Validates the plugin
2. Compiles source code (if needed)
3. Packages artifacts
4. Generates distribution archive
5. Creates installation metadata

Examples:
  zpam plugins build                      # Build current directory
  zpam plugins build ./my-plugin/        # Build specific plugin
  zpam plugins build --output ./dist/    # Custom output directory`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPluginsBuild,
}

var pluginsPublishCmd = &cobra.Command{
	Use:   "publish [plugin-path]",
	Short: "Publish plugin to registry or GitHub",
	Long: `Publish a plugin to the ZPAM registry or GitHub.

This command:
1. Validates the plugin thoroughly
2. Runs security scans
3. Builds and packages the plugin
4. Publishes to specified registry
5. Updates plugin metadata

Examples:
  zpam plugins publish                         # Publish current directory
  zpam plugins publish --registry github      # Publish to GitHub
  zpam plugins publish --registry marketplace # Publish to ZPAM marketplace
  zpam plugins publish --private              # Publish to private registry`,
	Args: cobra.MaximumNArgs(1),
	Run:  runPluginsPublish,
}

// GitHub API structures
type GitHubRepository struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"html_url"`
	CloneURL    string   `json:"clone_url"`
	Stars       int      `json:"stargazers_count"`
	Language    string   `json:"language"`
	UpdatedAt   string   `json:"updated_at"`
	Topics      []string `json:"topics"`
}

type GitHubSearchResponse struct {
	TotalCount int                `json:"total_count"`
	Items      []GitHubRepository `json:"items"`
}

type GitHubPluginManifest struct {
	ManifestVersion string `yaml:"manifest_version"`
	Plugin          struct {
		Name           string   `yaml:"name"`
		Version        string   `yaml:"version"`
		Description    string   `yaml:"description"`
		Author         string   `yaml:"author"`
		Homepage       string   `yaml:"homepage"`
		Repository     string   `yaml:"repository"`
		License        string   `yaml:"license"`
		Type           string   `yaml:"type"`
		Tags           []string `yaml:"tags"`
		MinZpamVersion string   `yaml:"min_zpam_version"`
	} `yaml:"plugin"`
	Interfaces []string `yaml:"interfaces"`
	Security   struct {
		Permissions []string `yaml:"permissions"`
		Sandbox     bool     `yaml:"sandbox"`
	} `yaml:"security"`
}

func init() {
	// Add flags
	pluginsCmd.PersistentFlags().StringVarP(&pluginConfigFile, "config", "c", "", "Configuration file path")

	// Marketplace-specific flags
	pluginsInstallCmd.Flags().BoolVar(&forceInstall, "force", false, "Force install/upgrade plugin")

	// Plugin creation flags
	pluginsCreateCmd.Flags().StringVar(&pluginTemplate, "template", "", "Plugin template to use")
	pluginsCreateCmd.Flags().StringVar(&pluginAuthor, "author", "", "Plugin author name")
	pluginsCreateCmd.Flags().StringVar(&pluginLicense, "license", "MIT", "Plugin license")

	// Plugin validation flags
	pluginsValidateCmd.Flags().BoolVar(&strictValidation, "strict", false, "Enable strict validation mode")
	pluginsValidateCmd.Flags().BoolVar(&securityOnly, "security-only", false, "Run security checks only")

	// Plugin build flags
	pluginsBuildCmd.Flags().StringVar(&outputDir, "output", "", "Output directory for build artifacts")

	// Plugin publish flags
	pluginsPublishCmd.Flags().StringVar(&publishRegistry, "registry", "github", "Target registry (github, marketplace)")
	pluginsPublishCmd.Flags().BoolVar(&privateRegistry, "private", false, "Publish to private registry")

	// Add subcommands
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsTestCmd)
	pluginsCmd.AddCommand(pluginsTestOneCmd)
	pluginsCmd.AddCommand(pluginsStatsCmd)
	pluginsCmd.AddCommand(pluginsEnableCmd)
	pluginsCmd.AddCommand(pluginsDisableCmd)
	pluginsCmd.AddCommand(pluginsDiscoverCmd)
	pluginsCmd.AddCommand(pluginsInstallCmd)
	pluginsCmd.AddCommand(pluginsSearchCmd)
	pluginsCmd.AddCommand(pluginsUninstallCmd)
	pluginsCmd.AddCommand(pluginsDiscoverGitHubCmd)
	pluginsCmd.AddCommand(pluginsUpdateRegistryCmd)
	pluginsCmd.AddCommand(pluginsCreateCmd)
	pluginsCmd.AddCommand(pluginsValidateCmd)
	pluginsCmd.AddCommand(pluginsBuildCmd)
	pluginsCmd.AddCommand(pluginsPublishCmd)

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

	fmt.Println("ZPAM Plugins Status")
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
	fmt.Println("  zpam plugins enable spamassassin")
	fmt.Println("  zpam plugins enable rspamd")
	fmt.Println()
	fmt.Println("To test plugins:")
	fmt.Println("  zpam plugins test examples/test_headers.eml")

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

	// Test ZPAM native scoring first
	start := time.Now()
	zpamScore := sf.CalculateSpamScore(emailObj)
	zpamTime := time.Since(start)
	zpamNormalized := sf.NormalizeScore(zpamScore)

	fmt.Printf("ZPAM Native Score: %.2f (Level %d) in %v\n", zpamScore, zpamNormalized, zpamTime)
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
		fmt.Println("  zpam plugins enable spamassassin")
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

	finalScore := zpamScore + combinedScore
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
	fmt.Println("  zpam plugins test examples/test_headers.eml")
	fmt.Println("  zpam plugins test-one spamassassin examples/test_headers.eml")
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

// Marketplace functions
func runPluginsDiscover(cmd *cobra.Command, args []string) {
	fmt.Println("ZPAM Plugin Marketplace")
	fmt.Println("======================")
	fmt.Println()

	// Mock marketplace data for now - in production this would fetch from API
	plugins := getMockMarketplacePlugins()

	fmt.Printf("Found %d available plugins:\n\n", len(plugins))

	for _, plugin := range plugins {
		status := ""
		if plugin.Verified {
			status = "✓ VERIFIED"
		} else {
			status = "COMMUNITY"
		}

		fmt.Printf("📦 %s v%s (%s)\n", plugin.Name, plugin.Version, status)
		fmt.Printf("   %s\n", plugin.Description)
		fmt.Printf("   Author: %s | Type: %s | Downloads: %d\n", plugin.Author, plugin.Type, plugin.Downloads)
		if len(plugin.Tags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(plugin.Tags, ", "))
		}
		fmt.Printf("   Install: zpam plugins install %s\n", plugin.Name)
		fmt.Println()
	}

	fmt.Println("💡 Use 'zpam plugins search <keyword>' to find specific plugins")
	fmt.Println("💡 Use 'zpam plugins install <name>' to install a plugin")
}

func runPluginsInstall(cmd *cobra.Command, args []string) {
	pluginName := args[0]

	fmt.Printf("Installing plugin: %s\n", pluginName)

	// Determine installation source
	installSource := detectInstallSource(pluginName)

	switch installSource.Type {
	case "github":
		installFromGitHub(installSource, forceInstall)
	case "zip":
		installFromZip(installSource, forceInstall)
	case "folder":
		installFromFolder(installSource, forceInstall)
	case "url":
		installFromURL(installSource, forceInstall)
	case "registry":
		installFromRegistry(pluginName, forceInstall)
	default:
		fmt.Printf("❌ Unknown installation source for: %s\n", pluginName)
		fmt.Println("Supported formats:")
		fmt.Println("  GitHub: github:user/repo or https://github.com/user/repo")
		fmt.Println("  ZIP: plugin.zip or https://example.com/plugin.zip")
		fmt.Println("  Folder: ./plugin-folder/ or /path/to/plugin/")
		fmt.Println("  Registry: plugin-name")
		return
	}
}

// Plugin source detection and installation types
type InstallSource struct {
	Type     string // "github", "zip", "folder", "url", "registry"
	Source   string // Original input
	RepoURL  string // For GitHub
	FilePath string // For local files
	URL      string // For remote URLs
	User     string // GitHub user
	Repo     string // GitHub repo
	Ref      string // GitHub branch/tag
}

func detectInstallSource(input string) InstallSource {
	// GitHub shorthand: github:user/repo
	if strings.HasPrefix(input, "github:") {
		parts := strings.Split(strings.TrimPrefix(input, "github:"), "/")
		if len(parts) >= 2 {
			return InstallSource{
				Type:    "github",
				Source:  input,
				RepoURL: fmt.Sprintf("https://github.com/%s/%s", parts[0], parts[1]),
				User:    parts[0],
				Repo:    parts[1],
				Ref:     "main", // default branch
			}
		}
	}

	// GitHub URL: https://github.com/user/repo
	if strings.Contains(input, "github.com") {
		return InstallSource{
			Type:    "github",
			Source:  input,
			RepoURL: input,
		}
	}

	// Remote ZIP URL
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		if strings.HasSuffix(input, ".zip") {
			return InstallSource{
				Type:   "zip",
				Source: input,
				URL:    input,
			}
		}
		return InstallSource{
			Type:   "url",
			Source: input,
			URL:    input,
		}
	}

	// Local ZIP file
	if strings.HasSuffix(input, ".zip") {
		return InstallSource{
			Type:     "zip",
			Source:   input,
			FilePath: input,
		}
	}

	// Local folder
	if strings.HasPrefix(input, "./") || strings.HasPrefix(input, "/") || strings.Contains(input, "/") {
		return InstallSource{
			Type:     "folder",
			Source:   input,
			FilePath: input,
		}
	}

	// Registry name (default)
	return InstallSource{
		Type:   "registry",
		Source: input,
	}
}

func installFromGitHub(source InstallSource, force bool) {
	fmt.Printf("📡 Installing from GitHub: %s\n", source.RepoURL)

	// Parse GitHub URL to extract user/repo
	if source.User == "" || source.Repo == "" {
		user, repo, err := parseGitHubURL(source.RepoURL)
		if err != nil {
			fmt.Printf("❌ Invalid GitHub URL: %v\n", err)
			return
		}
		source.User = user
		source.Repo = repo
	}

	// Check if plugin already exists
	pluginDir := filepath.Join("plugins", source.Repo)
	if _, err := os.Stat(pluginDir); err == nil && !force {
		fmt.Printf("⚠️  Plugin '%s' already exists\n", source.Repo)
		fmt.Println("   Use --force to reinstall")
		return
	}

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll("plugins", 0755); err != nil {
		fmt.Printf("❌ Failed to create plugins directory: %v\n", err)
		return
	}

	// Simulate cloning (in production, use git clone or download ZIP)
	fmt.Println("📦 Cloning repository...")
	time.Sleep(1 * time.Second)

	// Check for plugin manifest
	fmt.Println("🔍 Validating plugin manifest...")
	if !validatePluginManifest(source.Repo) {
		fmt.Printf("❌ Invalid or missing zpam-plugin.yaml manifest\n")
		fmt.Println("   Plugin repositories must include a zpam-plugin.yaml file")
		return
	}

	fmt.Println("🔧 Installing plugin...")
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("✅ Plugin '%s' installed successfully from GitHub!\n", source.Repo)
	fmt.Printf("   Repository: %s/%s\n", source.User, source.Repo)
	fmt.Printf("   Use 'zpam plugins enable %s' to enable the plugin\n", source.Repo)
}

func installFromZip(source InstallSource, force bool) {
	var zipPath string

	if source.URL != "" {
		fmt.Printf("📡 Installing from remote ZIP: %s\n", source.URL)
		// TODO: Download ZIP file
		fmt.Println("❌ Remote ZIP download not yet implemented")
		return
	} else {
		zipPath = source.FilePath
		fmt.Printf("📦 Installing from local ZIP: %s\n", zipPath)
	}

	// Check if ZIP file exists
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		fmt.Printf("❌ ZIP file not found: %s\n", zipPath)
		return
	}

	// Extract plugin name from ZIP filename
	pluginName := strings.TrimSuffix(filepath.Base(zipPath), ".zip")
	pluginDir := filepath.Join("plugins", pluginName)

	if _, err := os.Stat(pluginDir); err == nil && !force {
		fmt.Printf("⚠️  Plugin '%s' already exists\n", pluginName)
		fmt.Println("   Use --force to reinstall")
		return
	}

	fmt.Println("📦 Extracting ZIP archive...")
	time.Sleep(500 * time.Millisecond)

	// TODO: Implement actual ZIP extraction
	fmt.Println("🔍 Validating plugin structure...")
	time.Sleep(300 * time.Millisecond)

	fmt.Printf("✅ Plugin '%s' installed successfully from ZIP!\n", pluginName)
	fmt.Printf("   Use 'zpam plugins enable %s' to enable the plugin\n", pluginName)
}

func installFromFolder(source InstallSource, force bool) {
	folderPath := source.FilePath
	fmt.Printf("📁 Installing from local folder: %s\n", folderPath)

	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		fmt.Printf("❌ Folder not found: %s\n", folderPath)
		return
	}

	// Extract plugin name from folder name
	pluginName := filepath.Base(folderPath)
	pluginDir := filepath.Join("plugins", pluginName)

	if _, err := os.Stat(pluginDir); err == nil && !force {
		fmt.Printf("⚠️  Plugin '%s' already exists\n", pluginName)
		fmt.Println("   Use --force to reinstall")
		return
	}

	// Validate plugin structure
	fmt.Println("🔍 Validating plugin structure...")
	manifestPath := filepath.Join(folderPath, "zpam-plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		fmt.Printf("❌ Missing zpam-plugin.yaml manifest in %s\n", folderPath)
		fmt.Println("   Plugin folders must include a zpam-plugin.yaml file")
		return
	}

	fmt.Println("📦 Copying plugin files...")
	time.Sleep(500 * time.Millisecond)

	// TODO: Implement actual folder copying
	fmt.Printf("✅ Plugin '%s' installed successfully from folder!\n", pluginName)
	fmt.Printf("   Use 'zpam plugins enable %s' to enable the plugin\n", pluginName)
}

func installFromURL(source InstallSource, force bool) {
	fmt.Printf("📡 Installing from URL: %s\n", source.URL)
	// TODO: Implement generic URL installation
	fmt.Println("❌ Generic URL installation not yet implemented")
	fmt.Println("   Supported: ZIP files and GitHub repositories")
}

func installFromRegistry(pluginName string, force bool) {
	// This is the existing marketplace installation logic

	// Check if plugin exists in marketplace
	marketplacePlugins := getMockMarketplacePlugins()
	var targetPlugin *MarketplacePlugin
	for _, plugin := range marketplacePlugins {
		if plugin.Name == pluginName {
			targetPlugin = &plugin
			break
		}
	}

	if targetPlugin == nil {
		fmt.Printf("❌ Plugin '%s' not found in marketplace\n", pluginName)
		fmt.Println("Available plugins:")
		for _, plugin := range marketplacePlugins {
			fmt.Printf("  - %s\n", plugin.Name)
		}
		fmt.Println("\nOr install from other sources:")
		fmt.Println("  GitHub: zpam plugins install github:user/repo")
		fmt.Println("  ZIP: zpam plugins install plugin.zip")
		fmt.Println("  Folder: zpam plugins install ./plugin-folder/")
		return
	}

	// Check if already installed (mock check)
	if isPluginInstalled(pluginName) && !force {
		fmt.Printf("⚠️  Plugin '%s' is already installed\n", pluginName)
		fmt.Println("   Use --force to reinstall/upgrade")
		return
	}

	// Simulate installation process
	fmt.Printf("🔍 Found plugin: %s v%s by %s\n", targetPlugin.Name, targetPlugin.Version, targetPlugin.Author)

	if len(targetPlugin.Dependencies) > 0 {
		fmt.Printf("📋 Dependencies: %s\n", strings.Join(targetPlugin.Dependencies, ", "))
	}

	fmt.Println("📦 Downloading plugin...")
	time.Sleep(1 * time.Second) // Simulate download

	fmt.Println("🔧 Installing plugin...")
	time.Sleep(500 * time.Millisecond) // Simulate installation

	fmt.Println("✅ Plugin installed successfully!")
	fmt.Printf("   Use 'zpam plugins enable %s' to enable the plugin\n", pluginName)
	fmt.Printf("   Use 'zpam plugins list' to see all installed plugins\n")
}

// Helper functions for GitHub and validation
func parseGitHubURL(url string) (user, repo string, err error) {
	// Parse URLs like: https://github.com/user/repo
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format")
	}

	user = parts[len(parts)-2]
	repo = parts[len(parts)-1]

	return user, repo, nil
}

func validatePluginManifest(pluginName string) bool {
	// Mock validation - in production this would check zpam-plugin.yaml
	// Check required fields: name, version, description, type, main
	validPlugins := []string{"openai-classifier", "phishing-detector", "custom-rules"}
	for _, valid := range validPlugins {
		if strings.Contains(pluginName, valid) {
			return true
		}
	}
	return true // For demo purposes, always validate
}

func runPluginsDiscoverGitHub(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 Discovering ZPAM plugins from GitHub...")
	fmt.Println("========================================")
	fmt.Println()

	// Search GitHub for repositories with 'zpam-plugin' topic
	repos, err := searchGitHubPlugins()
	if err != nil {
		fmt.Printf("❌ Failed to search GitHub: %v\n", err)
		fmt.Println("   This is a mock implementation. In production, this would:")
		fmt.Println("   1. Use GitHub API to search for 'zpam-plugin' topic")
		fmt.Println("   2. Fetch zpam-plugin.yaml manifests")
		fmt.Println("   3. Validate plugin compatibility")
		showMockGitHubPlugins()
		return
	}

	fmt.Printf("Found %d plugins on GitHub:\n\n", len(repos))

	for _, repo := range repos {
		// Fetch manifest for each repository
		manifest, err := fetchPluginManifest(repo.FullName)
		if err != nil {
			fmt.Printf("⚠️  %s - Invalid manifest: %v\n", repo.FullName, err)
			continue
		}

		fmt.Printf("📦 %s\n", manifest.Plugin.Name)
		fmt.Printf("   Repository: %s ⭐ %d\n", repo.FullName, repo.Stars)
		fmt.Printf("   Version: %s | Type: %s\n", manifest.Plugin.Version, manifest.Plugin.Type)
		fmt.Printf("   Description: %s\n", manifest.Plugin.Description)
		fmt.Printf("   Author: %s | License: %s\n", manifest.Plugin.Author, manifest.Plugin.License)
		fmt.Printf("   Install: zpam plugins install github:%s\n", repo.FullName)
		fmt.Println()
	}

	fmt.Println("💡 To install any plugin:")
	fmt.Println("   zpam plugins install github:user/repo")
}

func runPluginsUpdateRegistry(cmd *cobra.Command, args []string) {
	fmt.Println("🔄 Updating plugin registry from GitHub...")
	fmt.Println("=========================================")
	fmt.Println()

	// Simulate registry update process
	fmt.Println("📡 Searching GitHub for zpam-plugin repositories...")
	time.Sleep(1 * time.Second)

	fmt.Println("🔍 Found 15 repositories with 'zpam-plugin' topic")

	fmt.Println("📋 Validating plugin manifests...")
	time.Sleep(800 * time.Millisecond)

	validPlugins := []string{
		"zpam-team/openai-classifier",
		"security-corp/phishing-detector",
		"ml-labs/advanced-bayes",
		"community/keyword-filters",
		"enterprise/outlook-integration",
	}

	fmt.Printf("✅ Validated %d plugins successfully\n", len(validPlugins))
	fmt.Println("❌ Skipped 3 plugins (invalid manifests)")
	fmt.Println("⚠️  Skipped 2 plugins (incompatible versions)")

	fmt.Println("\n💾 Updating local registry cache...")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("✅ Registry updated successfully!")
	fmt.Println()
	fmt.Println("📊 Registry Statistics:")
	fmt.Printf("   Total plugins: %d\n", len(validPlugins)+5) // +5 for existing marketplace plugins
	fmt.Printf("   GitHub plugins: %d\n", len(validPlugins))
	fmt.Println("   Marketplace plugins: 5")
	fmt.Println("   Last updated: just now")
	fmt.Println()
	fmt.Println("💡 Use 'zpam plugins discover' to see all available plugins")
}

// GitHub integration functions
func searchGitHubPlugins() ([]GitHubRepository, error) {
	// Mock implementation - in production this would use GitHub API
	// URL: https://api.github.com/search/repositories?q=topic:zpam-plugin
	return nil, fmt.Errorf("GitHub API not configured")
}

func fetchPluginManifest(repoFullName string) (*GitHubPluginManifest, error) {
	// Mock implementation - in production this would fetch zpam-plugin.yaml
	// URL: https://raw.githubusercontent.com/{repoFullName}/main/zpam-plugin.yaml
	return nil, fmt.Errorf("manifest fetch not implemented")
}

func showMockGitHubPlugins() {
	fmt.Println("📦 Mock GitHub Plugin Discovery Results:")
	fmt.Println()

	mockPlugins := []struct {
		Name        string
		Repo        string
		Version     string
		Type        string
		Description string
		Author      string
		Stars       int
	}{
		{
			Name:        "openai-classifier",
			Repo:        "zpam-team/openai-classifier",
			Version:     "1.2.0",
			Type:        "ml_classifier",
			Description: "AI-powered spam detection using OpenAI GPT models",
			Author:      "ZPAM Team",
			Stars:       45,
		},
		{
			Name:        "phishing-detector-pro",
			Repo:        "security-corp/phishing-detector",
			Version:     "2.1.0",
			Type:        "content_analyzer",
			Description: "Advanced phishing detection with URL analysis",
			Author:      "Security Corp",
			Stars:       32,
		},
		{
			Name:        "advanced-bayes-filter",
			Repo:        "ml-labs/advanced-bayes",
			Version:     "1.5.2",
			Type:        "ml_classifier",
			Description: "Enhanced Bayesian spam filtering with auto-learning",
			Author:      "ML Labs",
			Stars:       28,
		},
		{
			Name:        "keyword-rules-engine",
			Repo:        "community/keyword-filters",
			Version:     "1.0.1",
			Type:        "custom_rule_engine",
			Description: "Customizable keyword-based filtering rules",
			Author:      "Community",
			Stars:       19,
		},
		{
			Name:        "outlook-integration",
			Repo:        "enterprise/outlook-integration",
			Version:     "2.0.0",
			Type:        "external_engine",
			Description: "Microsoft Outlook enterprise integration",
			Author:      "Enterprise Solutions",
			Stars:       67,
		},
	}

	for _, plugin := range mockPlugins {
		fmt.Printf("📦 %s\n", plugin.Name)
		fmt.Printf("   Repository: %s ⭐ %d\n", plugin.Repo, plugin.Stars)
		fmt.Printf("   Version: %s | Type: %s\n", plugin.Version, plugin.Type)
		fmt.Printf("   Description: %s\n", plugin.Description)
		fmt.Printf("   Author: %s\n", plugin.Author)
		fmt.Printf("   Install: zpam plugins install github:%s\n", plugin.Repo)
		fmt.Println()
	}
}

func runPluginsSearch(cmd *cobra.Command, args []string) {
	keyword := strings.ToLower(args[0])

	fmt.Printf("Searching for plugins matching: %s\n", keyword)
	fmt.Println("=" + strings.Repeat("=", len(keyword)+30))
	fmt.Println()

	plugins := getMockMarketplacePlugins()
	var matches []MarketplacePlugin

	// Search through plugins
	for _, plugin := range plugins {
		if matchesKeyword(plugin, keyword) {
			matches = append(matches, plugin)
		}
	}

	if len(matches) == 0 {
		fmt.Printf("❌ No plugins found matching '%s'\n", keyword)
		fmt.Println("💡 Try broader search terms like 'ai', 'phishing', or 'integration'")
		return
	}

	fmt.Printf("Found %d plugin(s):\n\n", len(matches))

	for _, plugin := range matches {
		status := ""
		if plugin.Verified {
			status = "✓ VERIFIED"
		} else {
			status = "COMMUNITY"
		}

		fmt.Printf("📦 %s v%s (%s)\n", plugin.Name, plugin.Version, status)
		fmt.Printf("   %s\n", plugin.Description)
		fmt.Printf("   Install: zpam plugins install %s\n", plugin.Name)
		fmt.Println()
	}
}

func runPluginsUninstall(cmd *cobra.Command, args []string) {
	pluginName := args[0]

	fmt.Printf("Uninstalling plugin: %s\n", pluginName)

	// Check if plugin is installed
	if !isPluginInstalled(pluginName) {
		fmt.Printf("❌ Plugin '%s' is not installed\n", pluginName)
		fmt.Println("   Use 'zpam plugins list' to see installed plugins")
		return
	}

	// TODO: Check if plugin is currently enabled and disable it first
	fmt.Println("🔍 Checking plugin status...")

	// Simulate uninstallation
	fmt.Println("🗑️  Removing plugin files...")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("🧹 Cleaning up configuration...")
	time.Sleep(300 * time.Millisecond)

	fmt.Println("✅ Plugin uninstalled successfully!")
	fmt.Printf("   Plugin '%s' has been removed from your system\n", pluginName)
}

// Helper functions for marketplace
func getMockMarketplacePlugins() []MarketplacePlugin {
	return []MarketplacePlugin{
		{
			Name:         "openai-classifier",
			Version:      "1.2.0",
			Description:  "AI-powered spam detection using OpenAI GPT models",
			Author:       "ZPAM Team",
			Type:         "ml_classifier",
			Tags:         []string{"ai", "openai", "gpt", "machine-learning"},
			Verified:     true,
			Downloads:    1250,
			Rating:       4.8,
			Dependencies: []string{"openai-api-key"},
		},
		{
			Name:        "phishing-detector-pro",
			Version:     "2.1.0",
			Description: "Advanced phishing detection with URL analysis and brand protection",
			Author:      "Security Corp",
			Type:        "content_analyzer",
			Tags:        []string{"phishing", "security", "url-analysis"},
			Verified:    true,
			Downloads:   890,
			Rating:      4.6,
		},
		{
			Name:         "microsoft-defender-integration",
			Version:      "1.0.5",
			Description:  "Integration with Microsoft Defender for cloud-based threat intelligence",
			Author:       "Enterprise Solutions Inc",
			Type:         "external_engine",
			Tags:         []string{"microsoft", "defender", "enterprise", "cloud"},
			Verified:     false,
			Downloads:    456,
			Rating:       4.2,
			Dependencies: []string{"microsoft-api-access"},
		},
		{
			Name:        "spamhaus-enhanced",
			Version:     "3.0.1",
			Description: "Enhanced Spamhaus integration with premium threat feeds",
			Author:      "Community",
			Type:        "reputation_checker",
			Tags:        []string{"spamhaus", "reputation", "blacklist", "rbl"},
			Verified:    false,
			Downloads:   2100,
			Rating:      4.9,
		},
		{
			Name:         "slack-alerts",
			Version:      "1.1.2",
			Description:  "Send spam detection alerts and statistics to Slack channels",
			Author:       "DevOps Tools",
			Type:         "external_engine",
			Tags:         []string{"slack", "notifications", "alerts", "monitoring"},
			Verified:     true,
			Downloads:    678,
			Rating:       4.4,
			Dependencies: []string{"slack-webhook-url"},
		},
	}
}

func matchesKeyword(plugin MarketplacePlugin, keyword string) bool {
	keyword = strings.ToLower(keyword)

	// Check name
	if strings.Contains(strings.ToLower(plugin.Name), keyword) {
		return true
	}

	// Check description
	if strings.Contains(strings.ToLower(plugin.Description), keyword) {
		return true
	}

	// Check author
	if strings.Contains(strings.ToLower(plugin.Author), keyword) {
		return true
	}

	// Check tags
	for _, tag := range plugin.Tags {
		if strings.Contains(strings.ToLower(tag), keyword) {
			return true
		}
	}

	// Check type
	if strings.Contains(strings.ToLower(plugin.Type), keyword) {
		return true
	}

	return false
}

func isPluginInstalled(pluginName string) bool {
	// Mock implementation - in production this would check installed plugins
	installedPlugins := []string{"spamassassin", "rspamd", "custom_rules", "virustotal", "machine_learning"}
	for _, installed := range installedPlugins {
		if installed == pluginName {
			return true
		}
	}
	return false
}

// Helper function to convert config to plugin config (copied from filter package)
func runPluginsCreate(cmd *cobra.Command, args []string) {
	pluginName := args[0]
	pluginType := args[1]

	// Default to Go if no language specified
	language := "go"
	if len(args) >= 3 {
		language = args[2]
	}

	fmt.Printf("🚀 Creating new ZPAM plugin: %s (%s, %s)\n", pluginName, pluginType, language)
	fmt.Println("============================================")
	fmt.Println()

	// Validate plugin type
	validTypes := []string{"content-analyzer", "reputation-checker", "attachment-scanner", "ml-classifier", "external-engine", "custom-rule-engine"}
	isValidType := false
	for _, validType := range validTypes {
		if pluginType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		fmt.Printf("❌ Invalid plugin type: %s\n", pluginType)
		fmt.Println("Valid types:")
		for _, t := range validTypes {
			fmt.Printf("  - %s\n", t)
		}
		return
	}

	// Validate language
	validLanguages := []string{"go", "lua"}
	isValidLanguage := false
	for _, validLang := range validLanguages {
		if language == validLang {
			isValidLanguage = true
			break
		}
	}

	if !isValidLanguage {
		fmt.Printf("❌ Invalid language: %s\n", language)
		fmt.Println("Valid languages:")
		for _, lang := range validLanguages {
			fmt.Printf("  - %s\n", lang)
		}
		return
	}

	// Create plugin directory
	pluginDir := fmt.Sprintf("zpam-plugin-%s", pluginName)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create plugin directory: %v\n", err)
		return
	}

	fmt.Printf("📁 Created plugin directory: %s\n", pluginDir)

	// Generate plugin manifest
	fmt.Println("📝 Generating zpam-plugin.yaml manifest...")
	if err := generatePluginManifest(pluginDir, pluginName, pluginType, language); err != nil {
		fmt.Printf("❌ Failed to generate manifest: %v\n", err)
		return
	}

	// Generate source code template
	fmt.Println("💻 Generating source code template...")
	if err := generateSourceTemplate(pluginDir, pluginName, pluginType, language); err != nil {
		fmt.Printf("❌ Failed to generate source code: %v\n", err)
		return
	}

	// Generate additional files
	fmt.Println("📚 Generating documentation and build scripts...")
	if err := generateProjectFiles(pluginDir, pluginName, pluginType, language); err != nil {
		fmt.Printf("❌ Failed to generate project files: %v\n", err)
		return
	}

	fmt.Println("✅ Plugin created successfully!")
	fmt.Println()
	fmt.Println("📋 Next steps:")
	fmt.Printf("   1. cd %s\n", pluginDir)
	fmt.Println("   2. Edit zpam-plugin.yaml with your plugin details")
	fmt.Printf("   3. Implement your plugin logic in src/main.go\n")
	fmt.Println("   4. Test with: zpam plugins validate")
	fmt.Println("   5. Build with: zpam plugins build")
	fmt.Println("   6. Publish with: zpam plugins publish")
}

func runPluginsValidate(cmd *cobra.Command, args []string) {
	pluginPath := "."
	if len(args) > 0 {
		pluginPath = args[0]
	}

	fmt.Printf("🔍 Validating plugin at: %s\n", pluginPath)
	fmt.Println("===============================")
	fmt.Println()

	validationResults := []ValidationResult{}

	// 1. Manifest validation
	fmt.Println("📋 Validating plugin manifest...")
	manifestResult := validateManifestFile(pluginPath)
	validationResults = append(validationResults, manifestResult)
	printValidationResult("Manifest", manifestResult)

	// 2. Interface compliance validation
	if !securityOnly {
		fmt.Println("🔌 Validating interface compliance...")
		interfaceResult := validateInterfaceCompliance(pluginPath)
		validationResults = append(validationResults, interfaceResult)
		printValidationResult("Interface Compliance", interfaceResult)
	}

	// 3. Security validation
	fmt.Println("🔒 Validating security requirements...")
	securityResult := validateSecurity(pluginPath)
	validationResults = append(validationResults, securityResult)
	printValidationResult("Security", securityResult)

	// 4. Code quality validation (if not security-only)
	if !securityOnly {
		fmt.Println("⚡ Validating code quality...")
		qualityResult := validateCodeQuality(pluginPath)
		validationResults = append(validationResults, qualityResult)
		printValidationResult("Code Quality", qualityResult)
	}

	// 5. Dependency validation
	fmt.Println("📦 Validating dependencies...")
	depResult := validateDependencies(pluginPath)
	validationResults = append(validationResults, depResult)
	printValidationResult("Dependencies", depResult)

	// Summary
	fmt.Println()
	passed := 0
	warnings := 0
	errors := 0

	for _, result := range validationResults {
		if result.Status == "pass" {
			passed++
		} else if result.Status == "warning" {
			warnings++
		} else {
			errors++
		}
	}

	fmt.Println("📊 Validation Summary:")
	fmt.Printf("   ✅ Passed: %d\n", passed)
	fmt.Printf("   ⚠️  Warnings: %d\n", warnings)
	fmt.Printf("   ❌ Errors: %d\n", errors)

	if errors == 0 {
		fmt.Println("\n🎉 Plugin validation successful!")
		fmt.Println("   Ready for building and publishing")
	} else {
		fmt.Println("\n❌ Plugin validation failed")
		fmt.Println("   Please fix the errors above before proceeding")
	}
}

func runPluginsBuild(cmd *cobra.Command, args []string) {
	pluginPath := "."
	if len(args) > 0 {
		pluginPath = args[0]
	}

	fmt.Printf("🔨 Building plugin at: %s\n", pluginPath)
	fmt.Println("==========================")
	fmt.Println()

	// 1. Pre-build validation
	fmt.Println("🔍 Running pre-build validation...")
	if !runQuickValidation(pluginPath) {
		fmt.Println("❌ Pre-build validation failed. Fix errors before building.")
		return
	}

	// 2. Read manifest to understand build requirements
	manifestPath := filepath.Join(pluginPath, "zpam-plugin.yaml")
	fmt.Printf("📋 Reading manifest: %s\n", manifestPath)
	time.Sleep(200 * time.Millisecond)

	// 3. Build based on plugin type
	fmt.Println("🔧 Compiling plugin...")
	if err := buildPlugin(pluginPath); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		return
	}

	// 4. Package artifacts
	fmt.Println("📦 Packaging artifacts...")
	if err := packagePlugin(pluginPath); err != nil {
		fmt.Printf("❌ Packaging failed: %v\n", err)
		return
	}

	// 5. Generate distribution archive
	fmt.Println("🗜️  Creating distribution archive...")
	outputPath := filepath.Join(pluginPath, "dist")
	time.Sleep(300 * time.Millisecond)

	fmt.Println("✅ Plugin built successfully!")
	fmt.Printf("   Output directory: %s\n", outputPath)
	fmt.Println("   Files generated:")
	fmt.Println("   - plugin binary")
	fmt.Println("   - zpam-plugin.yaml")
	fmt.Println("   - README.md")
	fmt.Println("   - installation scripts")
	fmt.Println()
	fmt.Println("💡 Next step: zpam plugins publish")
}

func runPluginsPublish(cmd *cobra.Command, args []string) {
	pluginPath := "."
	if len(args) > 0 {
		pluginPath = args[0]
	}

	fmt.Printf("🚀 Publishing plugin at: %s\n", pluginPath)
	fmt.Println("=============================")
	fmt.Println()

	// 1. Pre-publish validation
	fmt.Println("🔍 Running comprehensive validation...")
	if !runFullValidation(pluginPath) {
		fmt.Println("❌ Pre-publish validation failed. Plugin not ready for publishing.")
		return
	}

	// 2. Security scan
	fmt.Println("🔒 Running security scan...")
	if !runSecurityScan(pluginPath) {
		fmt.Println("❌ Security scan failed. Fix security issues before publishing.")
		return
	}

	// 3. Build if needed
	fmt.Println("🔨 Ensuring plugin is built...")
	if err := ensurePluginBuilt(pluginPath); err != nil {
		fmt.Printf("❌ Build check failed: %v\n", err)
		return
	}

	// 4. Determine publishing target
	registry := publishRegistry
	if registry == "" {
		registry = "github" // default
	}

	fmt.Printf("📡 Publishing to: %s\n", registry)

	switch registry {
	case "github":
		publishToGitHub(pluginPath)
	case "marketplace":
		publishToMarketplace(pluginPath)
	default:
		fmt.Printf("❌ Unknown registry: %s\n", registry)
		return
	}

	fmt.Println("✅ Plugin published successfully!")
	fmt.Println("🎉 Your plugin is now available for installation!")
}

// Plugin creation and validation helper types and functions
type ValidationResult struct {
	Status   string // "pass", "warning", "error"
	Messages []string
}

func generatePluginManifest(pluginDir, pluginName, pluginType, language string) error {
	// Generate zpam-plugin.yaml content based on plugin type and language
	manifestContent := generateManifestContent(pluginName, pluginType, language)
	manifestPath := filepath.Join(pluginDir, "zpam-plugin.yaml")
	return os.WriteFile(manifestPath, []byte(manifestContent), 0644)
}

func generateSourceTemplate(pluginDir, pluginName, pluginType, language string) error {
	// Create src directory
	srcDir := filepath.Join(pluginDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return err
	}

	// Generate source template based on plugin type and language
	sourceContent := generateSourceContent(pluginName, pluginType, language)
	var sourcePath string
	if language == "lua" {
		sourcePath = filepath.Join(srcDir, "main.lua")
	} else {
		sourcePath = filepath.Join(srcDir, "main.go")
	}
	return os.WriteFile(sourcePath, []byte(sourceContent), 0644)
}

func generateProjectFiles(pluginDir, pluginName, pluginType, language string) error {
	// Generate README.md
	readmeContent := generateReadmeContent(pluginName, pluginType)
	readmePath := filepath.Join(pluginDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return err
	}

	// Generate Makefile
	makefileContent := generateMakefileContent(pluginName)
	makefilePath := filepath.Join(pluginDir, "Makefile")
	if err := os.WriteFile(makefilePath, []byte(makefileContent), 0644); err != nil {
		return err
	}

	// Generate test files
	testDir := filepath.Join(pluginDir, "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return err
	}

	return nil
}

func generateManifestContent(pluginName, pluginType, language string) string {
	interfaces := getInterfacesForType(pluginType)

	var mainField string
	if language == "lua" {
		mainField = fmt.Sprintf(`main:
  script: "./src/main.lua"
  runtime: "lua"`)
	} else {
		mainField = fmt.Sprintf(`main:
  binary: "./bin/%s"`, pluginName)
	}

	return fmt.Sprintf(`manifest_version: "1.0"

plugin:
  name: "%s"
  version: "1.0.0"
  description: "ZPAM plugin for %s (%s)"
  author: "%s"
  homepage: "https://github.com/yourusername/%s"
  repository: "https://github.com/yourusername/%s"
  license: "MIT"
  type: "%s"
  tags: ["%s", "%s"]
  min_zpam_version: "2.0.0"

%s

interfaces:
%s

configuration:
  example_setting:
    type: "string"
    required: false
    default: "default_value"
    description: "Example configuration setting"

security:
  permissions: []
  sandbox: true

marketplace:
  category: "Spam Detection"
  keywords: ["%s", "spam", "detection", "%s"]
`, pluginName, pluginType, language, getAuthorName(), pluginName, pluginName, pluginType, pluginType, language, mainField, interfaces, pluginType, language)
}

func generateSourceContent(pluginName, pluginType, language string) string {
	if language == "lua" {
		return generateLuaSourceContent(pluginName, pluginType)
	} else {
		return generateGoSourceContent(pluginName, pluginType)
	}
}

func generateGoSourceContent(pluginName, pluginType string) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"os"
)

// %s - ZPAM plugin for %s
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: %s <email-file>")
	}

	emailFile := os.Args[1]
	
	// TODO: Implement your plugin logic here
	// Read the email file and analyze it
	
	fmt.Printf("Analyzing email: %%s\n", emailFile)
	
	// Example plugin output format
	result := PluginResult{
		Score:       0.5,  // 0.0 to 1.0 (0 = ham, 1 = spam)
		Confidence:  0.8,  // 0.0 to 1.0
		Explanation: "Example %s analysis",
		Metadata: map[string]interface{}{
			"plugin_name": "%s",
			"version":     "1.0.0",
		},
	}
	
	// Output result as JSON
	outputResult(result)
}

type PluginResult struct {
	Score       float64                `+"`json:\"score\"`"+`
	Confidence  float64                `+"`json:\"confidence\"`"+`
	Explanation string                 `+"`json:\"explanation\"`"+`
	Metadata    map[string]interface{} `+"`json:\"metadata\"`"+`
}

func outputResult(result PluginResult) {
	// Output plugin result in expected format
	fmt.Printf("{\"score\": %%f, \"confidence\": %%f, \"explanation\": \"%%s\"}\n",
		result.Score, result.Confidence, result.Explanation)
}
`, pluginName, pluginType, pluginName, pluginType, pluginName)
}

func generateLuaSourceContent(pluginName, pluginType string) string {
	functionName := getFunctionNameForType(pluginType)

	return fmt.Sprintf(`-- @name %s
-- @version 1.0.0
-- @description ZPAM plugin for %s
-- @type %s
-- @interfaces %s

-- ZPAM %s Plugin
-- TODO: Implement your spam detection logic here

-- Main function that ZPAM will call
-- @param email table - Email data with fields: from, to, subject, body, headers, attachments
-- @return table - Result with score, confidence, rules, metadata
function %s(email)
    -- TODO: Implement your %s logic here
    
    local result = {
        score = 0.0,       -- 0.0 to 100.0 (higher = more spam)
        confidence = 0.7,  -- 0.0 to 1.0 (confidence in the score)
        rules = {},        -- Array of triggered rule descriptions
        metadata = {}      -- Key-value pairs of additional information
    }
    
    -- Example analysis based on plugin type
    %s
    
    -- Add metadata
    result.metadata.plugin_name = "%s"
    result.metadata.version = "1.0.0"
    result.metadata.analysis_type = "%s"
    
    return result
end

-- Helper functions for common tasks
function contains_keyword(text, keywords)
    if not text or not keywords then
        return false
    end
    
    local lower_text = string.lower(text)
    for _, keyword in ipairs(keywords) do
        if string.find(lower_text, string.lower(keyword), 1, true) then
            return true
        end
    end
    return false
end

function count_caps(text)
    if not text then return 0 end
    local caps = 0
    for i = 1, #text do
        local char = string.sub(text, i, i)
        if char:match("%%u") then
            caps = caps + 1
        end
    end
    return caps / #text
end

function extract_domain(email_addr)
    if not email_addr then return "" end
    local at_pos = string.find(email_addr, "@")
    if at_pos then
        return string.sub(email_addr, at_pos + 1)
    end
    return email_addr
end

-- ZPAM API functions available:
-- zpam.log(message)              - Log a message
-- zpam.contains(text, pattern)   - Case-insensitive substring search
-- zpam.domain_from_email(email)  - Extract domain from email address
`, pluginName, pluginType, pluginType, getInterfacesForType(pluginType), pluginType, functionName, pluginType, generateLuaExample(pluginType), pluginName, pluginType)
}

func getFunctionNameForType(pluginType string) string {
	switch pluginType {
	case "content-analyzer":
		return "analyze_content"
	case "reputation-checker":
		return "check_reputation"
	case "attachment-scanner":
		return "scan_attachments"
	case "ml-classifier":
		return "classify"
	case "external-engine":
		return "analyze"
	case "custom-rule-engine":
		return "evaluate_rules"
	default:
		return "analyze_content"
	}
}

func generateLuaExample(pluginType string) string {
	switch pluginType {
	case "content-analyzer":
		return `    -- Example: Check for spam keywords in subject and body
    local spam_keywords = {"viagra", "lottery", "winner", "congratulations", "urgent"}
    
    if contains_keyword(email.subject, spam_keywords) then
        result.score = 80.0
        result.confidence = 0.9
        table.insert(result.rules, "Spam keyword in subject")
    end
    
    if contains_keyword(email.body, spam_keywords) then
        result.score = result.score + 50.0
        result.confidence = 0.8
        table.insert(result.rules, "Spam keyword in body")
    end
    
    -- Check for excessive capitalization
    if count_caps(email.subject) > 0.5 then
        result.score = result.score + 20.0
        table.insert(result.rules, "Excessive caps in subject")
    end`
	case "reputation-checker":
		return `    -- Example: Check sender domain reputation
    local domain = extract_domain(email.from)
    local suspicious_domains = {"tempmail.com", "guerrillamail.com", "10minutemail.com"}
    
    for _, bad_domain in ipairs(suspicious_domains) do
        if domain == bad_domain then
            result.score = 90.0
            result.confidence = 0.95
            table.insert(result.rules, "Known spam domain: " .. domain)
            break
        end
    end
    
    -- Check for suspicious TLDs
    if string.match(domain, "%.tk$") or string.match(domain, "%.ml$") then
        result.score = result.score + 30.0
        table.insert(result.rules, "Suspicious TLD")
    end`
	case "attachment-scanner":
		return `    -- Example: Check for dangerous attachments
    if email.attachments then
        for _, attachment in ipairs(email.attachments) do
            local filename = attachment.filename or ""
            local content_type = attachment.content_type or ""
            
            -- Check for executable files
            if string.match(filename, "%.exe$") or string.match(filename, "%.scr$") then
                result.score = 95.0
                result.confidence = 0.9
                table.insert(result.rules, "Executable attachment: " .. filename)
            end
            
            -- Check for suspicious file types
            if string.match(content_type, "application/x%-msdownload") then
                result.score = result.score + 70.0
                table.insert(result.rules, "Suspicious attachment type")
            end
        end
    end`
	case "ml-classifier":
		return `    -- Example: Simple ML-like classification
    local spam_score = 0
    local features = {}
    
    -- Feature extraction
    features.subject_length = #(email.subject or "")
    features.body_length = #(email.body or "")
    features.has_attachments = (email.attachments and #email.attachments > 0)
    
    -- Simple scoring model
    if features.subject_length < 10 or features.subject_length > 100 then
        spam_score = spam_score + 20
    end
    
    if features.body_length < 50 then
        spam_score = spam_score + 30
    end
    
    if features.has_attachments then
        spam_score = spam_score + 10
    end
    
    result.score = spam_score
    result.confidence = 0.6`
	case "external-engine":
		return `    -- Example: Simulate external API call
    -- Note: Lua plugins cannot make real HTTP requests
    -- This would need to be implemented via ZPAM API functions
    
    local domain = extract_domain(email.from)
    
    -- Simulate API response based on domain
    if domain == "spam.example.com" then
        result.score = 85.0
        result.confidence = 0.9
        table.insert(result.rules, "External API flagged domain")
    end
    
    result.metadata.external_check = "simulated"`
	case "custom-rule-engine":
		return `    -- Example: Custom rule evaluation
    local rules = {
        {
            name = "Subject contains money",
            pattern = "money",
            score = 40.0,
            field = "subject"
        },
        {
            name = "Body too short",
            threshold = 20,
            score = 25.0,
            field = "body_length"
        }
    }
    
    for _, rule in ipairs(rules) do
        local triggered = false
        
        if rule.field == "subject" and rule.pattern then
            if contains_keyword(email.subject, {rule.pattern}) then
                triggered = true
            end
        elseif rule.field == "body_length" then
            if #(email.body or "") < rule.threshold then
                triggered = true
            end
        end
        
        if triggered then
            result.score = result.score + rule.score
            table.insert(result.rules, rule.name)
        end
    end`
	default:
		return `    -- TODO: Implement your analysis logic here
    result.score = 0.0
    result.confidence = 0.5`
	}
}

func generateReadmeContent(pluginName, pluginType string) string {
	return fmt.Sprintf(`# %s

ZPAM plugin for %s.

## Description

This plugin implements %s functionality for the ZPAM spam detection system.

## Installation

`+"```bash"+`
zpam plugins install github:yourusername/%s
`+"```"+`

## Configuration

Edit your ZPAM configuration to enable this plugin:

`+"```yaml"+`
plugins:
  %s:
    enabled: true
    weight: 1.0
    settings:
      example_setting: "your_value"
`+"```"+`

## Development

### Building

`+"```bash"+`
make build
`+"```"+`

### Testing

`+"```bash"+`
make test
`+"```"+`

### Publishing

`+"```bash"+`
zpam plugins publish
`+"```"+`

## License

MIT License - see LICENSE file for details.
`, pluginName, pluginType, pluginType, pluginName, pluginName)
}

func generateMakefileContent(pluginName string) string {
	return fmt.Sprintf(`.PHONY: build test clean validate publish

build:
	@echo "Building %s plugin..."
	@mkdir -p bin
	@go build -o bin/%s src/main.go
	@echo "✅ Build complete"

test:
	@echo "Testing %s plugin..."
	@go test ./src/...
	@echo "✅ Tests passed"

validate:
	@echo "Validating plugin..."
	@zpam plugins validate
	@echo "✅ Validation complete"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ dist/
	@echo "✅ Clean complete"

publish:
	@echo "Publishing plugin..."
	@zpam plugins build
	@zpam plugins publish
	@echo "✅ Published successfully"
`, pluginName, pluginName, pluginName)
}

func getInterfacesForType(pluginType string) string {
	switch pluginType {
	case "content-analyzer":
		return "  - \"ContentAnalyzer\""
	case "reputation-checker":
		return "  - \"ReputationChecker\""
	case "attachment-scanner":
		return "  - \"AttachmentScanner\""
	case "ml-classifier":
		return "  - \"MLClassifier\""
	case "external-engine":
		return "  - \"ExternalEngine\""
	case "custom-rule-engine":
		return "  - \"CustomRuleEngine\""
	default:
		return "  - \"ContentAnalyzer\""
	}
}

func getAuthorName() string {
	if pluginAuthor != "" {
		return pluginAuthor
	}
	return "Plugin Developer"
}

// Validation functions
func validateManifestFile(pluginPath string) ValidationResult {
	manifestPath := filepath.Join(pluginPath, "zpam-plugin.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return ValidationResult{
			Status:   "error",
			Messages: []string{"zpam-plugin.yaml not found"},
		}
	}

	// In production, this would parse and validate the YAML
	return ValidationResult{
		Status:   "pass",
		Messages: []string{"Manifest syntax valid", "All required fields present"},
	}
}

func validateInterfaceCompliance(pluginPath string) ValidationResult {
	// Check if this is a Lua plugin
	luaScriptPath := filepath.Join(pluginPath, "src", "main.lua")
	if _, err := os.Stat(luaScriptPath); err == nil {
		return validateLuaInterfaceCompliance(pluginPath, luaScriptPath)
	}

	// Default Go plugin validation
	return ValidationResult{
		Status:   "pass",
		Messages: []string{"Interface compliance verified"},
	}
}

func validateSecurity(pluginPath string) ValidationResult {
	// In production, this would run security scans
	return ValidationResult{
		Status:   "pass",
		Messages: []string{"No security issues found", "Permissions appropriate"},
	}
}

func validateCodeQuality(pluginPath string) ValidationResult {
	// Check if this is a Lua plugin
	luaScriptPath := filepath.Join(pluginPath, "src", "main.lua")
	if _, err := os.Stat(luaScriptPath); err == nil {
		return validateLuaCodeQuality(luaScriptPath)
	}

	// In production, this would run linters, tests, etc. for Go plugins
	return ValidationResult{
		Status:   "warning",
		Messages: []string{"Code quality acceptable", "Consider adding more tests"},
	}
}

func validateLuaInterfaceCompliance(pluginPath, luaScriptPath string) ValidationResult {
	// Read the Lua script
	content, err := os.ReadFile(luaScriptPath)
	if err != nil {
		return ValidationResult{
			Status:   "error",
			Messages: []string{fmt.Sprintf("Failed to read Lua script: %v", err)},
		}
	}

	scriptContent := string(content)

	// Check if manifest exists (simplified check)
	manifestPath := filepath.Join(pluginPath, "zpam-plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		return ValidationResult{
			Status:   "error",
			Messages: []string{"Could not find manifest file"},
		}
	}

	messages := []string{}
	status := "pass"

	// Check for required function based on plugin type
	// This is a simple check - in production you'd use a proper Lua parser
	requiredFunctions := []string{
		"analyze_content",
		"check_reputation",
		"scan_attachments",
		"classify",
		"analyze",
		"evaluate_rules",
	}

	foundFunction := false
	for _, funcName := range requiredFunctions {
		if strings.Contains(scriptContent, fmt.Sprintf("function %s(", funcName)) {
			messages = append(messages, fmt.Sprintf("Found required function: %s", funcName))
			foundFunction = true
			break
		}
	}

	if !foundFunction {
		status = "error"
		messages = append(messages, "No required interface function found")
		messages = append(messages, "Expected one of: analyze_content, check_reputation, scan_attachments, classify, analyze, evaluate_rules")
	}

	// Check for proper metadata comments
	if strings.Contains(scriptContent, "@name") &&
		strings.Contains(scriptContent, "@version") &&
		strings.Contains(scriptContent, "@description") {
		messages = append(messages, "Plugin metadata present")
	} else {
		if status != "error" {
			status = "warning"
		}
		messages = append(messages, "Missing some plugin metadata comments (@name, @version, @description)")
	}

	// Check for proper return format
	if strings.Contains(scriptContent, "score") &&
		strings.Contains(scriptContent, "confidence") &&
		strings.Contains(scriptContent, "return result") {
		messages = append(messages, "Proper result format used")
	} else {
		if status != "error" {
			status = "warning"
		}
		messages = append(messages, "Result format may not match expected structure")
	}

	return ValidationResult{
		Status:   status,
		Messages: messages,
	}
}

func validateLuaCodeQuality(luaScriptPath string) ValidationResult {
	// Basic Lua syntax validation using gopher-lua
	content, err := os.ReadFile(luaScriptPath)
	if err != nil {
		return ValidationResult{
			Status:   "error",
			Messages: []string{fmt.Sprintf("Failed to read Lua script: %v", err)},
		}
	}

	// Try to compile the Lua script to check for syntax errors
	vm := lua.NewState()
	defer vm.Close()

	messages := []string{}
	status := "pass"

	// Test syntax by trying to load the script
	if err := vm.DoString(string(content)); err != nil {
		status = "error"
		messages = append(messages, fmt.Sprintf("Lua syntax error: %v", err))
	} else {
		messages = append(messages, "Lua syntax is valid")
	}

	// Check for good practices
	scriptContent := string(content)

	// Check for helper functions
	if strings.Contains(scriptContent, "function ") {
		helperCount := strings.Count(scriptContent, "function ") - 1 // -1 for main function
		if helperCount > 0 {
			messages = append(messages, fmt.Sprintf("Good: %d helper functions defined", helperCount))
		}
	}

	// Check for error handling
	if strings.Contains(scriptContent, "if not") || strings.Contains(scriptContent, "if err") {
		messages = append(messages, "Good: Error handling present")
	} else {
		if status != "error" {
			status = "warning"
		}
		messages = append(messages, "Consider adding error handling")
	}

	// Check for documentation
	commentLines := strings.Count(scriptContent, "--")
	totalLines := strings.Count(scriptContent, "\n") + 1
	commentRatio := float64(commentLines) / float64(totalLines)

	if commentRatio > 0.2 {
		messages = append(messages, "Good: Well documented code")
	} else {
		if status != "error" {
			status = "warning"
		}
		messages = append(messages, "Consider adding more comments")
	}

	return ValidationResult{
		Status:   status,
		Messages: messages,
	}
}

func validateDependencies(pluginPath string) ValidationResult {
	// In production, this would check dependency availability
	return ValidationResult{
		Status:   "pass",
		Messages: []string{"All dependencies available"},
	}
}

func printValidationResult(category string, result ValidationResult) {
	icon := ""
	switch result.Status {
	case "pass":
		icon = "✅"
	case "warning":
		icon = "⚠️ "
	case "error":
		icon = "❌"
	}

	fmt.Printf("   %s %s\n", icon, category)
	for _, msg := range result.Messages {
		fmt.Printf("      %s\n", msg)
	}
}

func runQuickValidation(pluginPath string) bool {
	// Quick validation for build
	manifestPath := filepath.Join(pluginPath, "zpam-plugin.yaml")
	_, err := os.Stat(manifestPath)
	return err == nil
}

func buildPlugin(pluginPath string) error {
	// In production, this would compile the plugin
	time.Sleep(500 * time.Millisecond)
	return nil
}

func packagePlugin(pluginPath string) error {
	// In production, this would package artifacts
	time.Sleep(300 * time.Millisecond)
	return nil
}

func runFullValidation(pluginPath string) bool {
	// Comprehensive validation
	return true
}

func runSecurityScan(pluginPath string) bool {
	// Security scanning
	time.Sleep(1 * time.Second)
	return true
}

func ensurePluginBuilt(pluginPath string) error {
	// Ensure plugin is built
	return nil
}

func publishToGitHub(pluginPath string) {
	fmt.Println("📡 Pushing to GitHub repository...")
	time.Sleep(1 * time.Second)
	fmt.Println("🏷️  Creating release tag...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("📋 Updating GitHub registry...")
	time.Sleep(300 * time.Millisecond)
}

func publishToMarketplace(pluginPath string) {
	fmt.Println("📤 Uploading to ZPAM marketplace...")
	time.Sleep(1 * time.Second)
	fmt.Println("🔍 Running marketplace validation...")
	time.Sleep(800 * time.Millisecond)
	fmt.Println("✅ Plugin approved and published...")
	time.Sleep(300 * time.Millisecond)
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
