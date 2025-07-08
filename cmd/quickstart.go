package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/email"
	"github.com/zpo/spam-filter/pkg/filter"
)

var (
	quickstartSkipTests bool
	quickstartForce     bool
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Interactive setup wizard for quick ZPO configuration",
	Long: `Get ZPO running in under 5 minutes with optimal configuration.

This command will:
1. Auto-detect your system capabilities (Redis, etc.)
2. Generate an optimal configuration file
3. Test ZPO with sample emails
4. Show you how to use ZPO effectively

Perfect for first-time users who want to see ZPO in action immediately!`,
	RunE: runQuickstart,
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	fmt.Printf("🫏 ZPO Interactive Quickstart\n")
	fmt.Printf("════════════════════════════════════════════════\n")
	fmt.Printf("Interactive setup wizard - get running in 5 minutes!\n\n")

	// Check if automated install was already run
	if _, err := os.Stat("config-quickstart.yaml"); err == nil && !quickstartForce {
		fmt.Printf("✅ Found existing ZPO configuration (config-quickstart.yaml)\n")
		fmt.Printf("💡 For fully automated setup, use: ./zpo install\n\n")
		fmt.Printf("Would you like to run a demo with the existing configuration? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(strings.TrimSpace(response)) == "y" {
			return runQuickDemo()
		}
		if !quickstartForce {
			fmt.Printf("💡 Use --force to reconfigure, or './zpo install --force' for automated setup\n")
			return nil
		}
	}

	// Show install option
	fmt.Printf("💡 Pro tip: For fully automated setup, use './zpo install'\n")
	fmt.Printf("This interactive mode gives you more control over configuration.\n\n")

	// Step 1: System Detection
	fmt.Printf("🔍 Step 1: Detecting System Capabilities...\n")
	capabilities := detectSystemCapabilities()
	printCapabilities(capabilities)

	// Step 2: Generate Config
	fmt.Printf("\n⚙️ Step 2: Generating Optimal Configuration...\n")
	configPath := "config-quickstart.yaml"

	cfg := generateQuickstartConfig(capabilities)
	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}
	fmt.Printf("✅ Configuration saved: %s\n", configPath)

	// Step 3: Test with Sample Emails (unless skipped)
	if !quickstartSkipTests {
		fmt.Printf("\n🧪 Step 3: Testing with Sample Emails...\n")
		if err := runSampleTests(cfg); err != nil {
			fmt.Printf("⚠️  Sample tests failed: %v\n", err)
			fmt.Printf("💡 ZPO is still configured and ready to use!\n")
		}

		// Offer training if available
		if hasQuickstartTrainingData() {
			fmt.Printf("\n🧠 Training data detected! Would you like to run training? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(strings.TrimSpace(response)) == "y" {
				runQuickTraining(configPath)
			}
		}
	}

	// Step 4: Show Next Steps
	fmt.Printf("\n🚀 Step 4: You're Ready to Go!\n")
	printNextSteps(configPath, capabilities)

	return nil
}

func runQuickDemo() error {
	fmt.Printf("🎬 Running ZPO demo with existing configuration...\n\n")

	// Load existing config
	cfg, err := config.LoadConfig("config-quickstart.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Run quick tests
	fmt.Printf("🧪 Testing ZPO with sample emails...\n")
	if err := runSampleTests(cfg); err != nil {
		fmt.Printf("⚠️  Demo failed: %v\n", err)
	}

	// Show commands to try
	fmt.Printf("\n🚀 Try these commands:\n")
	samples := findSampleEmails()
	if len(samples) > 0 {
		fmt.Printf("  ./zpo test %s --config config-quickstart.yaml\n", samples[0])
	}
	fmt.Printf("  ./zpo status --config config-quickstart.yaml\n")
	fmt.Printf("  ./zpo monitor --config config-quickstart.yaml\n")

	return nil
}

func runQuickTraining(configPath string) {
	fmt.Printf("  🚀 Running quick training...\n")

	// Use the enhanced training system
	if _, err := os.Stat("training-data"); err == nil {
		cmd := exec.Command("./zpo", "train", "--auto-discover", "training-data", "--config", configPath, "--quiet", "--reset")
		if err := cmd.Run(); err != nil {
			fmt.Printf("    ⚠️  Training failed: %v\n", err)
		} else {
			fmt.Printf("    ✅ Training completed\n")
		}
	}
}

func hasQuickstartTrainingData() bool {
	locations := []string{"training-data", "examples", "milter/emails"}
	for _, location := range locations {
		if files, err := filepath.Glob(filepath.Join(location, "*.eml")); err == nil && len(files) > 0 {
			return true
		}
	}
	return false
}

// SystemCapabilities holds detected system information
type SystemCapabilities struct {
	HasRedis     bool
	RedisURL     string
	HasDocker    bool
	HasSamples   bool
	SampleEmails []string
	WorkingDir   string
}

func detectSystemCapabilities() *SystemCapabilities {
	caps := &SystemCapabilities{
		WorkingDir: ".",
	}

	// Check for Redis
	fmt.Printf("  🔍 Checking Redis availability...")
	if isRedisAvailable() {
		caps.HasRedis = true
		caps.RedisURL = "redis://localhost:6379"
		fmt.Printf(" ✅ Found (localhost:6379)\n")
	} else {
		fmt.Printf(" ❌ Not available\n")
	}

	// Check for Docker
	fmt.Printf("  🔍 Checking Docker availability...")
	if isDockerAvailable() {
		caps.HasDocker = true
		fmt.Printf(" ✅ Found\n")
	} else {
		fmt.Printf(" ❌ Not available\n")
	}

	// Check for sample emails
	fmt.Printf("  🔍 Checking sample emails...")
	samples := findSampleEmails()
	if len(samples) > 0 {
		caps.HasSamples = true
		caps.SampleEmails = samples
		fmt.Printf(" ✅ Found %d samples\n", len(samples))
	} else {
		fmt.Printf(" ❌ No samples found\n")
	}

	return caps
}

func printCapabilities(caps *SystemCapabilities) {
	fmt.Printf("\n📊 System Capabilities Summary:\n")
	if caps.HasRedis {
		fmt.Printf("  ✅ Redis: Available for high-performance Bayesian learning\n")
	} else {
		fmt.Printf("  📁 Redis: Not available (will use file-based learning)\n")
	}

	if caps.HasDocker {
		fmt.Printf("  ✅ Docker: Available for containerized deployment\n")
	} else {
		fmt.Printf("  📦 Docker: Not available (native deployment only)\n")
	}

	if caps.HasSamples {
		fmt.Printf("  ✅ Sample Emails: %d available for immediate testing\n", len(caps.SampleEmails))
	} else {
		fmt.Printf("  📧 Sample Emails: None found (will create basic examples)\n")
	}
}

func generateQuickstartConfig(caps *SystemCapabilities) *config.Config {
	cfg := config.DefaultConfig()

	// Optimize based on capabilities
	if caps.HasRedis {
		// Use Redis backend for better performance
		cfg.Learning.Enabled = true
		cfg.Learning.Backend = "redis"
		cfg.Learning.Redis.RedisURL = caps.RedisURL
		cfg.Learning.Redis.KeyPrefix = "zpo:quickstart"
		cfg.Learning.Redis.MinLearns = 5 // Lower threshold for testing
		fmt.Printf("  🧠 Configured Redis Bayesian learning (high performance)\n")
	} else {
		// Use file backend as fallback
		cfg.Learning.Enabled = true
		cfg.Learning.Backend = "file"
		cfg.Learning.File.ModelPath = "zpo-quickstart-model.json"
		fmt.Printf("  📁 Configured file-based learning (fallback mode)\n")
	}

	// Optimize performance settings
	cfg.Performance.MaxConcurrentEmails = 4
	cfg.Performance.TimeoutMs = 5000
	cfg.Performance.CacheSize = 1000
	fmt.Printf("  ⚡ Optimized performance settings\n")

	// Set reasonable thresholds for beginners
	cfg.Detection.SpamThreshold = 4 // Clear spam only
	fmt.Printf("  🎯 Set balanced spam threshold (4/5)\n")

	return cfg
}

func runSampleTests(cfg *config.Config) error {
	spamFilter := filter.NewSpamFilterWithConfig(cfg)
	if spamFilter == nil {
		return fmt.Errorf("failed to create spam filter")
	}

	// Find or create sample emails
	samples := findSampleEmails()
	if len(samples) == 0 {
		fmt.Printf("  📧 No sample emails found, creating test examples...\n")
		samples = createTestEmails()
	}

	fmt.Printf("  🧪 Testing %d sample emails...\n", len(samples))

	start := time.Now()
	processed := 0
	var totalScore float64

	for i, samplePath := range samples {
		if i >= 5 { // Test max 5 samples for quick start
			break
		}

		parser := email.NewParser()
		parsedEmail, err := parser.ParseFromFile(samplePath)
		if err != nil {
			fmt.Printf("    ⚠️  Failed to parse %s: %v\n", filepath.Base(samplePath), err)
			continue
		}

		score := spamFilter.CalculateSpamScore(parsedEmail)
		normalizedScore := spamFilter.NormalizeScore(score)

		status := "✅ HAM"
		if normalizedScore >= cfg.Detection.SpamThreshold {
			status = "🚫 SPAM"
		}

		fmt.Printf("    %s - %s (score: %d/5)\n",
			filepath.Base(samplePath), status, normalizedScore)

		totalScore += score
		processed++
	}

	duration := time.Since(start)

	if processed > 0 {
		avgTime := float64(duration.Nanoseconds()) / float64(processed) / 1e6
		fmt.Printf("  ✅ Processed %d emails in %v (%.2fms per email)\n",
			processed, duration, avgTime)
		fmt.Printf("  📊 Average score: %.2f\n", totalScore/float64(processed))
	}

	return nil
}

func printNextSteps(configPath string, caps *SystemCapabilities) {
	fmt.Printf("════════════════════════════════════════════════\n")
	fmt.Printf("🎉 ZPO is now configured and ready!\n\n")

	fmt.Printf("📝 Quick Commands to Try:\n")
	fmt.Printf("  # Test a single email\n")
	fmt.Printf("  ./zpo test examples/clean_email.eml --config %s\n\n", configPath)

	fmt.Printf("  # Filter a directory of emails\n")
	fmt.Printf("  ./zpo filter --input your-emails/ --config %s\n\n", configPath)

	if caps.HasSamples {
		fmt.Printf("  # Train with your sample emails\n")
		fmt.Printf("  ./zpo train --spam-dir spam/ --ham-dir ham/ --config %s\n\n", configPath)
	}

	if caps.HasRedis {
		fmt.Printf("🧠 Redis Learning Enabled:\n")
		fmt.Printf("  - Your model will persist across restarts\n")
		fmt.Printf("  - Multiple ZPO instances can share learning\n")
		fmt.Printf("  - Performance optimized for high volume\n\n")
	}

	if caps.HasDocker {
		fmt.Printf("🐳 Docker Available:\n")
		fmt.Printf("  # Deploy with Docker for production\n")
		fmt.Printf("  docker-compose -f docker/docker-compose.yml up -d\n\n")
	}

	fmt.Printf("📚 Next Steps:\n")
	fmt.Printf("  1. Try the commands above with your own emails\n")
	fmt.Printf("  2. Check the documentation: docs/README.md\n")
	fmt.Printf("  3. Run performance benchmarks: ./testing/benchmark_simple.sh\n")
	if caps.HasRedis {
		fmt.Printf("  4. Train with more data to improve accuracy\n")
	}
	fmt.Printf("  5. Set up milter integration for real-time filtering\n\n")

	fmt.Printf("🆘 Need Help?\n")
	fmt.Printf("  - Run any command with --help for detailed usage\n")
	fmt.Printf("  - Check testing/README.md for troubleshooting\n")
	fmt.Printf("  - Visit the project documentation\n\n")

	fmt.Printf("🫏 Happy spam filtering! ZPO is ready to work like a reliable donkey.\n")
}

// Helper functions

func isRedisAvailable() bool {
	// Simple check - try executing redis-cli or check if Redis service is running
	cmd := exec.Command("redis-cli", "ping")
	return cmd.Run() == nil
}

func isDockerAvailable() bool {
	// Check if docker command is available and working
	cmd := exec.Command("docker", "version")
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	return err == nil
}

func findSampleEmails() []string {
	var samples []string

	// Check common sample locations
	sampleDirs := []string{
		"examples",
		"milter/emails",
		"test-data",
	}

	for _, dir := range sampleDirs {
		if files, err := filepath.Glob(filepath.Join(dir, "*.eml")); err == nil {
			samples = append(samples, files...)
		}
	}

	return samples
}

func createTestEmails() []string {
	// Create minimal test emails for demonstration
	testDir := "quickstart-test"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return nil
	}

	// Simple test emails
	emails := map[string]string{
		"test-clean.eml": `From: colleague@company.com
To: user@company.com
Subject: Project Update

Hi there,

The project is progressing well. Let's schedule a meeting next week.

Best regards,
Your Colleague`,

		"test-suspicious.eml": `From: noreply@suspicious-site.com
To: user@company.com
Subject: URGENT! Free Money Waiting!

CONGRATULATIONS! You've won $1,000,000!
Click here NOW to claim your prize!
Limited time offer!`,
	}

	var created []string
	for filename, content := range emails {
		path := filepath.Join(testDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err == nil {
			created = append(created, path)
		}
	}

	return created
}

func init() {
	quickstartCmd.Flags().BoolVar(&quickstartSkipTests, "skip-tests", false, "Skip sample email testing")
	quickstartCmd.Flags().BoolVar(&quickstartForce, "force", false, "Overwrite existing configuration")

	rootCmd.AddCommand(quickstartCmd)
}
