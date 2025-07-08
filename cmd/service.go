package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
)

var (
	serviceConfig  string
	serviceMode    string
	serviceDaemon  bool
	serviceForce   bool
	servicePidFile string
	serviceLogFile string
	serviceQuiet   bool
)

// Service management commands
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service management (start, stop, restart, reload)",
	Long: `Manage ZPO service lifecycle with commands:
- start: Start ZPO service in background
- stop: Gracefully stop running ZPO service  
- restart: Stop and start ZPO service
- reload: Reload configuration without restart

Supports both milter mode (Postfix integration) and standalone mode.`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start ZPO service",
	Long: `Start ZPO service in the background.

Modes:
- milter: Run as milter for Postfix/Sendmail integration
- standalone: Run standalone service for testing

The service will be started as a daemon unless --no-daemon is specified.`,
	RunE: runStart,
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop ZPO service",
	Long: `Gracefully stop the running ZPO service.

Uses SIGTERM for graceful shutdown, with optional --force flag for SIGKILL.
Automatically removes PID files and cleans up resources.`,
	RunE: runStop,
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart ZPO service",
	Long: `Stop and start ZPO service.

Performs graceful shutdown followed by startup with configuration validation.
Equivalent to running 'zpo stop' followed by 'zpo start'.`,
	RunE: runRestart,
}

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload ZPO configuration",
	Long: `Reload ZPO configuration without restarting the service.

Sends SIGHUP to the running process to reload configuration files.
Faster than restart and maintains existing connections.`,
	RunE: runReload,
}

// Service management functions

func runStart(cmd *cobra.Command, args []string) error {
	if !serviceQuiet {
		fmt.Printf("🫏 Starting ZPO service...\n")
	}

	// Check if already running
	if isServiceRunning() {
		pid, _ := getServicePID()
		return fmt.Errorf("ZPO service is already running (PID: %d)", pid)
	}

	// Validate configuration
	cfg, err := loadServiceConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	if !serviceQuiet {
		fmt.Printf("📋 Configuration: %s\n", getConfigFile())
		fmt.Printf("🔧 Mode: %s\n", serviceMode)
	}

	// Validate configuration
	if err := validateServiceConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %v", err)
	}

	// Start the service
	pid, err := startService(cfg)
	if err != nil {
		return fmt.Errorf("failed to start service: %v", err)
	}

	// Wait a moment and verify it's running
	time.Sleep(1 * time.Second)
	if !isServiceRunning() {
		return fmt.Errorf("service failed to start properly")
	}

	if !serviceQuiet {
		fmt.Printf("✅ ZPO service started successfully (PID: %d)\n", pid)
		fmt.Printf("📊 Check status: ./zpo status\n")
		fmt.Printf("📜 View logs: tail -f %s\n", getLogFile())
	}

	return nil
}

func runStop(cmd *cobra.Command, args []string) error {
	if !serviceQuiet {
		fmt.Printf("🫏 Stopping ZPO service...\n")
	}

	// Check if running
	if !isServiceRunning() {
		if !serviceQuiet {
			fmt.Printf("ℹ️  ZPO service is not running\n")
		}
		return nil
	}

	pid, err := getServicePID()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %v", err)
	}

	// Stop the service
	if err := stopService(pid); err != nil {
		return fmt.Errorf("failed to stop service: %v", err)
	}

	if !serviceQuiet {
		fmt.Printf("✅ ZPO service stopped successfully\n")
	}

	return nil
}

func runRestart(cmd *cobra.Command, args []string) error {
	if !serviceQuiet {
		fmt.Printf("🫏 Restarting ZPO service...\n")
	}

	// Stop if running
	if isServiceRunning() {
		if err := runStop(cmd, args); err != nil {
			return err
		}
		// Give it a moment to fully stop
		time.Sleep(2 * time.Second)
	}

	// Start the service
	return runStart(cmd, args)
}

func runReload(cmd *cobra.Command, args []string) error {
	if !serviceQuiet {
		fmt.Printf("🫏 Reloading ZPO configuration...\n")
	}

	// Check if running
	if !isServiceRunning() {
		return fmt.Errorf("ZPO service is not running")
	}

	pid, err := getServicePID()
	if err != nil {
		return fmt.Errorf("failed to get service PID: %v", err)
	}

	// Validate new configuration first
	cfg, err := loadServiceConfig()
	if err != nil {
		return fmt.Errorf("configuration error: %v", err)
	}

	if err := validateServiceConfig(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %v", err)
	}

	// Send SIGHUP to reload
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send reload signal: %v", err)
	}

	if !serviceQuiet {
		fmt.Printf("✅ Configuration reloaded successfully\n")
	}

	return nil
}

// Service management helpers

func isServiceRunning() bool {
	pid, err := getServicePID()
	if err != nil {
		return false
	}

	// Check if process exists and is actually ZPO
	process, err := os.FindProcess(pid)
	if err != nil {
		cleanupPIDFile()
		return false
	}

	// Try to send signal 0 to check if process exists
	if err := process.Signal(syscall.Signal(0)); err != nil {
		cleanupPIDFile()
		return false
	}

	return true
}

func getServicePID() (int, error) {
	pidFile := getPIDFile()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %v", err)
	}

	return pid, nil
}

func startService(cfg *config.Config) (int, error) {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("failed to get executable path: %v", err)
	}

	// Prepare command arguments
	args := []string{}

	// Add the service mode command
	switch serviceMode {
	case "milter":
		args = append(args, "milter")
	case "standalone":
		// For standalone mode, we'll use the milter command but with different config
		args = append(args, "milter")
	default:
		return 0, fmt.Errorf("unknown service mode: %s", serviceMode)
	}

	// Add configuration file
	if serviceConfig != "" {
		args = append(args, "--config", serviceConfig)
	}

	// Create command
	cmd := exec.Command(executable, args...)

	// Set up logging
	logFile := getLogFile()
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return 0, fmt.Errorf("failed to create log directory: %v", err)
	}

	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %v", err)
	}
	defer log.Close()

	cmd.Stdout = log
	cmd.Stderr = log

	// Start the process
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start process: %v", err)
	}

	pid := cmd.Process.Pid

	// Write PID file
	if err := writePIDFile(pid); err != nil {
		cmd.Process.Kill()
		return 0, fmt.Errorf("failed to write PID file: %v", err)
	}

	// If not daemon mode, wait for the process
	if !serviceDaemon {
		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Printf("\n🛑 Shutting down ZPO service...\n")
			cmd.Process.Signal(syscall.SIGTERM)
		}()

		// Wait for process to complete
		err := cmd.Wait()
		cleanupPIDFile()
		return pid, err
	}

	return pid, nil
}

func stopService(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		cleanupPIDFile()
		return fmt.Errorf("process not found: %v", err)
	}

	// Try graceful shutdown first
	if !serviceQuiet {
		fmt.Printf("🔄 Sending SIGTERM to PID %d...\n", pid)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		cleanupPIDFile()
		return fmt.Errorf("failed to send SIGTERM: %v", err)
	}

	// Wait for graceful shutdown
	gracefulTimeout := 10 * time.Second
	shutdownComplete := make(chan bool, 1)

	go func() {
		for i := 0; i < int(gracefulTimeout.Seconds()); i++ {
			if err := process.Signal(syscall.Signal(0)); err != nil {
				shutdownComplete <- true
				return
			}
			time.Sleep(1 * time.Second)
		}
		shutdownComplete <- false
	}()

	if graceful := <-shutdownComplete; graceful {
		if !serviceQuiet {
			fmt.Printf("✅ Service stopped gracefully\n")
		}
		cleanupPIDFile()
		return nil
	}

	// Force kill if requested or if graceful shutdown failed
	if serviceForce {
		if !serviceQuiet {
			fmt.Printf("⚡ Forcing shutdown with SIGKILL...\n")
		}
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to force kill: %v", err)
		}
	} else {
		return fmt.Errorf("service did not stop within %v, use --force to kill", gracefulTimeout)
	}

	cleanupPIDFile()
	return nil
}

func loadServiceConfig() (*config.Config, error) {
	configFile := getConfigFile()
	if configFile == "" {
		return config.DefaultConfig(), nil
	}
	return config.LoadConfig(configFile)
}

func validateServiceConfig(cfg *config.Config) error {
	// Basic validation
	if serviceMode == "milter" {
		if cfg.Milter.Address == "" {
			return fmt.Errorf("milter address not configured")
		}
		if cfg.Milter.ReadTimeoutMs == 0 {
			return fmt.Errorf("milter read timeout not configured")
		}
	}

	// Validate learning backend if enabled
	if cfg.Learning.Enabled {
		switch cfg.Learning.Backend {
		case "redis":
			if cfg.Learning.Redis.RedisURL == "" {
				return fmt.Errorf("Redis URL not configured for learning backend")
			}
		case "file":
			if cfg.Learning.File.ModelPath == "" {
				return fmt.Errorf("model file path not configured for file backend")
			}
		default:
			return fmt.Errorf("unknown learning backend: %s", cfg.Learning.Backend)
		}
	}

	return nil
}

func getConfigFile() string {
	if serviceConfig != "" {
		return serviceConfig
	}

	// Try to find a config file
	candidates := []string{"config-quickstart.yaml", "config.yaml", "config-redis.yaml"}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func getPIDFile() string {
	if servicePidFile != "" {
		return servicePidFile
	}
	return fmt.Sprintf("zpo-%s.pid", serviceMode)
}

func getLogFile() string {
	if serviceLogFile != "" {
		return serviceLogFile
	}
	return fmt.Sprintf("logs/zpo-%s.log", serviceMode)
}

func writePIDFile(pid int) error {
	pidFile := getPIDFile()
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
}

func cleanupPIDFile() {
	pidFile := getPIDFile()
	os.Remove(pidFile)
}

// Initialize service commands
func init() {
	// Add subcommands to service
	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
	serviceCmd.AddCommand(restartCmd)
	serviceCmd.AddCommand(reloadCmd)

	// Common flags for all service commands
	serviceCmd.PersistentFlags().StringVarP(&serviceConfig, "config", "c", "", "Configuration file path")
	serviceCmd.PersistentFlags().BoolVarP(&serviceQuiet, "quiet", "q", false, "Quiet output")

	// Start command flags
	startCmd.Flags().StringVarP(&serviceMode, "mode", "m", "milter", "Service mode: milter, standalone")
	startCmd.Flags().BoolVar(&serviceDaemon, "daemon", true, "Run as daemon (background process)")
	startCmd.Flags().StringVar(&servicePidFile, "pid-file", "", "Custom PID file path")
	startCmd.Flags().StringVar(&serviceLogFile, "log-file", "", "Custom log file path")

	// Stop command flags
	stopCmd.Flags().BoolVarP(&serviceForce, "force", "f", false, "Force kill if graceful shutdown fails")

	// Add service command to root
	rootCmd.AddCommand(serviceCmd)

	// Also add individual commands directly to root for convenience
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(reloadCmd)
}
