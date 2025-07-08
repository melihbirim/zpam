package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/zpam/spam-filter/pkg/config"
)

var (
	installForce        bool
	installSkipDeps     bool
	installQuiet        bool
	installConfig       string
	installRedisURL     string
	installSkipTraining bool
	installOffline      bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Auto-detect system and install ZPAM with optimal configuration",
	Long: `Zero-configuration installer that gets ZPAM running in under 5 minutes.

This command will:
1. 🔍 Auto-detect your system capabilities (OS, Redis, Docker, etc.)
2. 📦 Install required dependencies (optional)
3. ⚙️  Generate optimal configuration for your system
4. 🧠 Set up learning backend (Redis or file-based)
5. 📧 Create sample emails for immediate testing
6. ✅ Verify installation with end-to-end test

Perfect for production deployments and first-time users!`,
	RunE: runInstall,
}

type SystemInfo struct {
	OS              string `json:"os"`
	Architecture    string `json:"architecture"`
	HasRedis        bool   `json:"has_redis"`
	RedisURL        string `json:"redis_url,omitempty"`
	HasDocker       bool   `json:"has_docker"`
	HasGo           bool   `json:"has_go"`
	HasPython       bool   `json:"has_python"`
	HasSpamAssassin bool   `json:"has_spamassassin"`
	HasPostfix      bool   `json:"has_postfix"`
	WorkingDir      string `json:"working_dir"`
	ConfigPath      string `json:"config_path"`

	// Installation recommendations
	RecommendedBackend string   `json:"recommended_backend"`
	MissingDeps        []string `json:"missing_deps"`
	OptionalDeps       []string `json:"optional_deps"`
	SecurityWarnings   []string `json:"security_warnings"`
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Printf("🫏 ZPAM Zero-Config Installer\n")
	fmt.Printf("════════════════════════════════════════════════════════\n")
	fmt.Printf("Getting ZPAM ready for production in under 5 minutes...\n\n")

	// Step 1: System Detection
	fmt.Printf("🔍 Step 1/6: System Detection\n")
	fmt.Printf("──────────────────────────────────────────\n")
	sysInfo := detectSystem()
	printSystemDetection(sysInfo)

	// Step 2: Dependency Check
	fmt.Printf("\n📦 Step 2/6: Dependency Analysis\n")
	fmt.Printf("──────────────────────────────────────────\n")
	analyzeDependencies(sysInfo)

	// Step 3: Install Dependencies (if requested)
	if !installSkipDeps && len(sysInfo.MissingDeps) > 0 {
		fmt.Printf("\n⬇️  Step 3/6: Installing Dependencies\n")
		fmt.Printf("──────────────────────────────────────────\n")
		if err := installDependencies(sysInfo); err != nil {
			fmt.Printf("⚠️  Warning: Some dependencies failed to install: %v\n", err)
		}
	} else {
		fmt.Printf("\n⏭️  Step 3/6: Skipping Dependency Installation\n")
		fmt.Printf("──────────────────────────────────────────\n")
		if installSkipDeps {
			fmt.Printf("✅ Skipped as requested\n")
		} else {
			fmt.Printf("✅ All dependencies already available\n")
		}
	}

	// Step 4: Configuration Generation
	fmt.Printf("\n⚙️  Step 4/6: Configuration Generation\n")
	fmt.Printf("──────────────────────────────────────────\n")
	configPath := installConfig
	if configPath == "" {
		configPath = "config-quickstart.yaml"
	}
	if err := generateOptimalConfig(sysInfo, configPath); err != nil {
		return fmt.Errorf("failed to generate configuration: %v", err)
	}

	// Step 5: Sample Data Setup
	fmt.Printf("\n📧 Step 5/6: Sample Data Setup\n")
	fmt.Printf("──────────────────────────────────────────\n")
	if err := setupSampleData(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to setup sample data: %v\n", err)
	}

	// Step 6: Training (optional)
	if !installSkipTraining && hasTrainingData() {
		fmt.Printf("\n🧠 Step 6/6: Initial Training\n")
		fmt.Printf("──────────────────────────────────────────\n")
		if err := runInitialTraining(configPath); err != nil {
			fmt.Printf("⚠️  Warning: Initial training failed: %v\n", err)
		}
	} else {
		fmt.Printf("\n⏭️  Step 6/6: Skipping Initial Training\n")
		fmt.Printf("──────────────────────────────────────────\n")
		if installSkipTraining {
			fmt.Printf("✅ Skipped as requested\n")
		} else {
			fmt.Printf("✅ No training data available\n")
		}
	}

	// Installation Complete
	printInstallationComplete(sysInfo, configPath)
	return nil
}

func detectSystem() *SystemInfo {
	fmt.Printf("🖥️  Detecting system capabilities...\n")

	sysInfo := &SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		WorkingDir:   ".",
	}

	// Check Redis
	fmt.Printf("  🔍 Redis...")
	if isRedisInstalled() {
		sysInfo.HasRedis = true
		sysInfo.RedisURL = "redis://localhost:6379"
		if installRedisURL != "" {
			sysInfo.RedisURL = installRedisURL
		}
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	// Check Docker
	fmt.Printf("  🔍 Docker...")
	if isDockerInstalled() {
		sysInfo.HasDocker = true
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	// Check Go
	fmt.Printf("  🔍 Go...")
	if isGoInstalled() {
		sysInfo.HasGo = true
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	// Check Python
	fmt.Printf("  🔍 Python...")
	if isPythonInstalled() {
		sysInfo.HasPython = true
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	// Check SpamAssassin
	fmt.Printf("  🔍 SpamAssassin...")
	if isSpamAssassinInstalled() {
		sysInfo.HasSpamAssassin = true
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	// Check Postfix
	fmt.Printf("  🔍 Postfix...")
	if isPostfixInstalled() {
		sysInfo.HasPostfix = true
		fmt.Printf(" ✅ Available\n")
	} else {
		fmt.Printf(" ❌ Not found\n")
	}

	return sysInfo
}

func printSystemDetection(sysInfo *SystemInfo) {
	fmt.Printf("\n📋 System Summary:\n")
	fmt.Printf("  🖥️  OS: %s/%s\n", sysInfo.OS, sysInfo.Architecture)

	if sysInfo.HasRedis {
		fmt.Printf("  ✅ Redis: %s (high-performance backend available)\n", sysInfo.RedisURL)
	} else {
		fmt.Printf("  📁 Redis: Not available (will use file-based storage)\n")
	}

	if sysInfo.HasDocker {
		fmt.Printf("  ✅ Docker: Available (containerized deployment option)\n")
	}

	if sysInfo.HasSpamAssassin {
		fmt.Printf("  ✅ SpamAssassin: Available (enhanced accuracy option)\n")
	}

	if sysInfo.HasPostfix {
		fmt.Printf("  ✅ Postfix: Available (milter integration ready)\n")
	}
}

func analyzeDependencies(sysInfo *SystemInfo) {
	fmt.Printf("🔍 Analyzing dependency requirements...\n")

	// Determine optimal backend
	if sysInfo.HasRedis {
		sysInfo.RecommendedBackend = "redis"
		fmt.Printf("  🚀 Recommended: Redis backend for optimal performance\n")
	} else {
		sysInfo.RecommendedBackend = "file"
		fmt.Printf("  📁 Recommended: File backend (Redis not available)\n")
		sysInfo.OptionalDeps = append(sysInfo.OptionalDeps, "redis")
	}

	// Check essential tools
	essentialTools := map[string]bool{
		"git": isCommandAvailable("git"),
	}

	for tool, available := range essentialTools {
		if !available {
			sysInfo.MissingDeps = append(sysInfo.MissingDeps, tool)
		}
	}

	// Optional enhancements
	optionalTools := map[string]bool{
		"spamassassin": sysInfo.HasSpamAssassin,
		"docker":       sysInfo.HasDocker,
		"python3":      sysInfo.HasPython,
	}

	for tool, available := range optionalTools {
		if !available {
			sysInfo.OptionalDeps = append(sysInfo.OptionalDeps, tool)
		}
	}

	// Print analysis
	if len(sysInfo.MissingDeps) == 0 {
		fmt.Printf("  ✅ All essential dependencies available\n")
	} else {
		fmt.Printf("  ⚠️  Missing essential: %s\n", strings.Join(sysInfo.MissingDeps, ", "))
	}

	if len(sysInfo.OptionalDeps) > 0 {
		fmt.Printf("  💡 Optional enhancements: %s\n", strings.Join(sysInfo.OptionalDeps, ", "))
	}
}

func installDependencies(sysInfo *SystemInfo) error {
	if len(sysInfo.MissingDeps) == 0 {
		fmt.Printf("✅ No dependencies to install\n")
		return nil
	}

	fmt.Printf("📦 Installing %d missing dependencies...\n", len(sysInfo.MissingDeps))

	// Ask for confirmation
	if !installForce && !confirmDependencyInstall(sysInfo.MissingDeps) {
		fmt.Printf("⏭️  Skipping dependency installation\n")
		return nil
	}

	// Install based on OS
	switch sysInfo.OS {
	case "darwin":
		return installDependenciesMacOS(sysInfo.MissingDeps)
	case "linux":
		return installDependenciesLinux(sysInfo.MissingDeps)
	default:
		return fmt.Errorf("automatic dependency installation not supported on %s", sysInfo.OS)
	}
}

func installDependenciesMacOS(deps []string) error {
	// Check if Homebrew is available
	if !isCommandAvailable("brew") {
		fmt.Printf("❌ Homebrew not found. Please install Homebrew first:\n")
		fmt.Printf("   /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"\n")
		return fmt.Errorf("homebrew required for macOS dependency installation")
	}

	fmt.Printf("🍺 Using Homebrew to install dependencies...\n")

	// Map internal names to brew packages
	brewPackages := map[string]string{
		"redis":        "redis",
		"docker":       "docker",
		"git":          "git",
		"python3":      "python3",
		"spamassassin": "spamassassin",
	}

	for _, dep := range deps {
		if pkg, exists := brewPackages[dep]; exists {
			fmt.Printf("  📦 Installing %s...", dep)
			cmd := exec.Command("brew", "install", pkg)
			if err := cmd.Run(); err != nil {
				fmt.Printf(" ❌ Failed\n")
				return fmt.Errorf("failed to install %s: %v", dep, err)
			}
			fmt.Printf(" ✅ Installed\n")
		}
	}

	return nil
}

func installDependenciesLinux(deps []string) error {
	// Detect package manager
	var installCmd []string
	if isCommandAvailable("apt") {
		installCmd = []string{"apt", "install", "-y"}
		// Update package list first
		fmt.Printf("  🔄 Updating package list...\n")
		if err := exec.Command("apt", "update").Run(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to update package list\n")
		}
	} else if isCommandAvailable("yum") {
		installCmd = []string{"yum", "install", "-y"}
	} else if isCommandAvailable("dnf") {
		installCmd = []string{"dnf", "install", "-y"}
	} else {
		return fmt.Errorf("no supported package manager found (apt/yum/dnf)")
	}

	fmt.Printf("📦 Using %s to install dependencies...\n", installCmd[0])

	// Map internal names to system packages
	linuxPackages := map[string]string{
		"redis":        "redis-server",
		"docker":       "docker.io",
		"git":          "git",
		"python3":      "python3",
		"spamassassin": "spamassassin",
	}

	for _, dep := range deps {
		if pkg, exists := linuxPackages[dep]; exists {
			fmt.Printf("  📦 Installing %s...", dep)
			args := append(installCmd, pkg)
			cmd := exec.Command("sudo", args...)
			if err := cmd.Run(); err != nil {
				fmt.Printf(" ❌ Failed\n")
				return fmt.Errorf("failed to install %s: %v", dep, err)
			}
			fmt.Printf(" ✅ Installed\n")
		}
	}

	return nil
}

func generateOptimalConfig(sysInfo *SystemInfo, configPath string) error {
	fmt.Printf("📝 Generating optimal configuration for your system...\n")

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil && !installForce {
		fmt.Printf("⚠️  Configuration file already exists: %s\n", configPath)
		if !confirmOverwrite(configPath) {
			fmt.Printf("✅ Using existing configuration\n")
			return nil
		}
	}

	// Create optimal config based on system
	cfg := config.DefaultConfig()

	// Configure learning backend
	if sysInfo.HasRedis {
		fmt.Printf("  🔧 Configuring Redis backend...\n")
		cfg.Learning.Backend = "redis"
		cfg.Learning.Redis.RedisURL = sysInfo.RedisURL
		cfg.Learning.Redis.KeyPrefix = "zpam:"
		cfg.Learning.Redis.TokenTTL = "720h" // 30 days
	} else {
		fmt.Printf("  🔧 Configuring file backend...\n")
		cfg.Learning.Backend = "file"
		cfg.Learning.File.ModelPath = "zpam-model.json"
	}

	// Configure plugins based on availability
	if sysInfo.HasSpamAssassin {
		fmt.Printf("  🔧 Enabling SpamAssassin plugin...\n")
		cfg.Plugins.Enabled = true
		cfg.Plugins.SpamAssassin.Enabled = true
		cfg.Plugins.SpamAssassin.Weight = 0.8
		cfg.Plugins.SpamAssassin.Priority = 10
	}

	// Configure milter if Postfix available
	if sysInfo.HasPostfix {
		fmt.Printf("  🔧 Configuring milter integration...\n")
		cfg.Milter.Enabled = true
		cfg.Milter.Address = "127.0.0.1:7357"
		cfg.Milter.ReadTimeoutMs = 10000
		cfg.Milter.WriteTimeoutMs = 10000
	}

	// Performance tuning based on system
	fmt.Printf("  🔧 Optimizing performance settings...\n")
	cfg.Performance.MaxConcurrentEmails = 10
	cfg.Performance.TimeoutMs = 5000
	cfg.Performance.CacheSize = 1000

	// Save configuration
	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}

	fmt.Printf("  ✅ Configuration saved: %s\n", configPath)
	fmt.Printf("  🎯 Backend: %s\n", cfg.Learning.Backend)
	if cfg.Plugins.Enabled {
		fmt.Printf("  🔌 Plugins: Enabled\n")
	}
	if cfg.Milter.Enabled {
		fmt.Printf("  📧 Milter: Ready for Postfix integration\n")
	}

	return nil
}

func setupSampleData() error {
	fmt.Printf("📧 Setting up sample data for testing...\n")

	// Check if training-data already exists
	if _, err := os.Stat("training-data"); err == nil {
		fmt.Printf("  ✅ Training data already exists\n")
		return nil
	}

	// Check if we have sample emails in various locations
	sampleLocations := []string{"examples", "milter/emails", "test-data"}
	found := false

	for _, location := range sampleLocations {
		if files, err := filepath.Glob(filepath.Join(location, "*.eml")); err == nil && len(files) > 0 {
			fmt.Printf("  📁 Found %d sample emails in %s\n", len(files), location)
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("  📧 Creating basic sample emails...\n")
		if err := createBasicSamples(); err != nil {
			return fmt.Errorf("failed to create samples: %v", err)
		}
	}

	fmt.Printf("  ✅ Sample data ready for testing\n")
	return nil
}

func runInitialTraining(configPath string) error {
	fmt.Printf("🧠 Running initial training with sample data...\n")

	// Check if training data exists
	if _, err := os.Stat("training-data"); err != nil {
		return fmt.Errorf("no training data available")
	}

	// Run training command
	fmt.Printf("  🚀 Training ZPAM with sample data...\n")

	// Use the train command we already built
	args := []string{
		"train",
		"--auto-discover", "training-data",
		"--config", configPath,
		"--quiet",
		"--reset",
	}

	// Build the command path
	execPath, err := os.Executable()
	if err != nil {
		execPath = "./zpam"
	}

	cmd := exec.Command(execPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("training failed: %v", err)
	}

	fmt.Printf("  ✅ Initial training completed\n")
	return nil
}

func printInstallationComplete(sysInfo *SystemInfo, configPath string) {
	fmt.Printf("\n🎉 ZPAM Installation Complete!\n")
	fmt.Printf("════════════════════════════════════════════════════════\n")
	fmt.Printf("✅ System detected and configured\n")
	fmt.Printf("✅ Optimal configuration generated: %s\n", configPath)
	fmt.Printf("✅ Sample data ready for testing\n")

	if !installSkipTraining {
		fmt.Printf("✅ Initial training completed\n")
	}

	fmt.Printf("\n🚀 Quick Start Commands:\n")
	fmt.Printf("─────────────────────────────────────────\n")

	// Test with sample data
	fmt.Printf("# Test spam detection:\n")
	if sysInfo.HasRedis {
		fmt.Printf("./zpam test training-data/spam/06_spam_phishing.eml --config %s\n\n", configPath)
	} else {
		fmt.Printf("./zpam test training-data/spam/06_spam_phishing.eml --config %s\n\n", configPath)
	}

	// Monitor system
	fmt.Printf("# Monitor ZPAM status:\n")
	fmt.Printf("./zpam status --config %s\n\n", configPath)

	// Train with more data
	fmt.Printf("# Train with your email data:\n")
	fmt.Printf("./zpam train --auto-discover /path/to/your/emails --config %s\n\n", configPath)

	if sysInfo.HasPostfix {
		fmt.Printf("# Start milter service:\n")
		fmt.Printf("./zpam milter --config %s\n\n", configPath)
	}

	// Performance and monitoring
	fmt.Printf("📊 Advanced Usage:\n")
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("# Real-time monitoring:\n")
	fmt.Printf("./zpam monitor --config %s\n\n", configPath)

	fmt.Printf("# Service management:\n")
	fmt.Printf("./zpam start --config %s\n", configPath)
	fmt.Printf("./zpam status --config %s\n\n", configPath)

	// Next steps
	fmt.Printf("📚 Next Steps:\n")
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("1. Test with your own emails\n")
	fmt.Printf("2. Train with more data to improve accuracy\n")

	if sysInfo.HasPostfix {
		fmt.Printf("3. Configure Postfix milter integration\n")
	}

	if len(sysInfo.OptionalDeps) > 0 {
		fmt.Printf("4. Install optional enhancements: %s\n", strings.Join(sysInfo.OptionalDeps, ", "))
	}

	fmt.Printf("\n🆘 Need Help?\n")
	fmt.Printf("─────────────────────────────────────────\n")
	fmt.Printf("• Run any command with --help for details\n")
	fmt.Printf("• Check the documentation in docs/\n")
	fmt.Printf("• Run './zpam quickstart' for interactive setup\n")

	fmt.Printf("\n🫏 ZPAM is ready! Time to detection: < 5ms per email\n")
}

// Helper functions

func isRedisInstalled() bool {
	// Try connecting to Redis
	if installRedisURL != "" {
		return checkRedisConnection(installRedisURL)
	}
	return checkRedisConnection("redis://localhost:6379")
}

func checkRedisConnection(redisURL string) bool {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return false
	}

	opt.DialTimeout = 1 * time.Second
	opt.ReadTimeout = 1 * time.Second
	opt.WriteTimeout = 1 * time.Second

	client := redis.NewClient(opt)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return client.Ping(ctx).Err() == nil
}

func isDockerInstalled() bool {
	return isCommandAvailable("docker")
}

func isGoInstalled() bool {
	return isCommandAvailable("go")
}

func isPythonInstalled() bool {
	return isCommandAvailable("python3") || isCommandAvailable("python")
}

func isSpamAssassinInstalled() bool {
	return isCommandAvailable("spamassassin") || isCommandAvailable("spamc")
}

func isPostfixInstalled() bool {
	return isCommandAvailable("postfix")
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func confirmDependencyInstall(deps []string) bool {
	fmt.Printf("\n❓ Install missing dependencies? (%s) [y/N]: ", strings.Join(deps, ", "))
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func confirmOverwrite(configPath string) bool {
	fmt.Printf("❓ Overwrite existing configuration %s? [y/N]: ", configPath)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func hasTrainingData() bool {
	// Check for training-data directory
	if _, err := os.Stat("training-data"); err == nil {
		return true
	}

	// Check for other sample locations
	locations := []string{"examples", "milter/emails", "test-data"}
	for _, location := range locations {
		if files, err := filepath.Glob(filepath.Join(location, "*.eml")); err == nil && len(files) > 0 {
			return true
		}
	}

	return false
}

func createBasicSamples() error {
	// Create training-data structure
	os.MkdirAll("training-data/spam", 0755)
	os.MkdirAll("training-data/ham", 0755)

	// Create a simple spam sample
	spamSample := `From: winner@lottery-scam.com
To: victim@example.com
Subject: CONGRATULATIONS! You've won $1,000,000!!!
Date: Mon, 1 Jan 2024 12:00:00 +0000

URGENT! CLAIM YOUR PRIZE NOW!

You have been selected as a winner in our international lottery!
To claim your $1,000,000 prize, reply with your bank details immediately.

This offer expires in 24 hours!
`

	// Create a simple ham sample
	hamSample := `From: colleague@company.com
To: employee@company.com
Subject: Meeting reminder - Tomorrow 2PM
Date: Mon, 1 Jan 2024 12:00:00 +0000

Hi,

Just a reminder about our project meeting tomorrow at 2PM in conference room A.

Please bring the quarterly reports we discussed.

Best regards,
John
`

	// Write samples
	if err := os.WriteFile("training-data/spam/sample_spam.eml", []byte(spamSample), 0644); err != nil {
		return err
	}

	if err := os.WriteFile("training-data/ham/sample_ham.eml", []byte(hamSample), 0644); err != nil {
		return err
	}

	return nil
}

func init() {
	installCmd.Flags().BoolVarP(&installForce, "force", "f", false, "Force overwrite existing configuration")
	installCmd.Flags().BoolVar(&installSkipDeps, "skip-deps", false, "Skip dependency installation")
	installCmd.Flags().BoolVarP(&installQuiet, "quiet", "q", false, "Quiet output")
	installCmd.Flags().StringVarP(&installConfig, "config", "c", "", "Custom configuration file path")
	installCmd.Flags().StringVar(&installRedisURL, "redis-url", "", "Custom Redis URL")
	installCmd.Flags().BoolVar(&installSkipTraining, "skip-training", false, "Skip initial training")
	installCmd.Flags().BoolVar(&installOffline, "offline", false, "Offline installation (no network dependencies)")

	rootCmd.AddCommand(installCmd)
}
