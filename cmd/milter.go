package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/milter"
)

var (
	milterConfigFile string
	milterNetwork    string
	milterAddress    string
	milterDebug      bool
)

var milterCmd = &cobra.Command{
	Use:   "milter",
	Short: "Start milter server for Postfix/Sendmail integration",
	Long: `Start ZPO milter server to integrate with Postfix or Sendmail MTA.

The milter server listens on a socket (TCP or Unix) and processes incoming
emails in real-time as they are received by the MTA. This provides immediate
spam filtering without storing emails to disk.

Example usage:
  # Start milter server with default config
  zpo milter

  # Start milter server with custom config
  zpo milter --config /etc/zpo/milter.yaml

  # Start milter server on custom address
  zpo milter --network tcp --address 127.0.0.1:7357

  # Start milter server with debug logging
  zpo milter --debug

For Postfix integration, add to main.cf:
  smtpd_milters = inet:127.0.0.1:7357
  non_smtpd_milters = inet:127.0.0.1:7357
  milter_default_action = accept`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.LoadConfig(milterConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Override config with command line flags if provided
		if cmd.Flags().Changed("network") {
			cfg.Milter.Network = milterNetwork
		}
		if cmd.Flags().Changed("address") {
			cfg.Milter.Address = milterAddress
		}
		if milterDebug {
			cfg.Logging.Level = "debug"
		}

		// Enable milter if not already enabled
		cfg.Milter.Enabled = true

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("invalid configuration: %v", err)
		}

		// Create network listener
		listener, err := net.Listen(cfg.Milter.Network, cfg.Milter.Address)
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
		defer listener.Close()

		// Create milter server
		server, err := milter.NewServer(cfg)
		if err != nil {
			return fmt.Errorf("failed to create milter server: %v", err)
		}

		// Setup graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start server in goroutine
		serverErr := make(chan error, 1)
		go func() {
			fmt.Printf("🫏 ZPO Milter Server starting on %s://%s\n",
				cfg.Milter.Network, cfg.Milter.Address)
			fmt.Printf("📧 Ready to filter emails via milter protocol\n")
			fmt.Printf("⚡ Performance: max %d concurrent connections, %dms timeouts\n",
				cfg.Milter.MaxConcurrentConnections, cfg.Milter.ReadTimeoutMs)
			fmt.Printf("🎯 Thresholds: reject >= %d, quarantine >= %d\n",
				cfg.Milter.RejectThreshold, cfg.Milter.QuarantineThreshold)

			if milterConfigFile != "" {
				fmt.Printf("⚙️  Configuration: %s\n", milterConfigFile)
			}
			fmt.Printf("🚀 Press Ctrl+C to stop\n\n")

			serverErr <- server.Serve(ctx, listener)
		}()

		// Wait for shutdown signal or server error
		select {
		case <-sigChan:
			fmt.Printf("\n🛑 Shutdown signal received, stopping milter server...\n")

			// Create shutdown context with timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(
				context.Background(),
				time.Duration(cfg.Milter.GracefulShutdownTimeout)*time.Millisecond,
			)
			defer shutdownCancel()

			// Cancel server context to initiate shutdown
			cancel()

			// Wait for graceful shutdown or timeout
			select {
			case err := <-serverErr:
				if err != nil && err != context.Canceled {
					fmt.Printf("⚠️  Server shutdown with error: %v\n", err)
				} else {
					fmt.Printf("✅ Milter server stopped gracefully\n")
				}
			case <-shutdownCtx.Done():
				fmt.Printf("⏰ Shutdown timeout exceeded, forcing stop\n")
			}

		case err := <-serverErr:
			if err != nil {
				return fmt.Errorf("milter server error: %v", err)
			}
		}

		return nil
	},
}

func init() {
	milterCmd.Flags().StringVarP(&milterConfigFile, "config", "c", "config.yaml", "Configuration file path")
	milterCmd.Flags().StringVarP(&milterNetwork, "network", "n", "", "Network type (tcp or unix)")
	milterCmd.Flags().StringVarP(&milterAddress, "address", "a", "", "Bind address (e.g., 127.0.0.1:7357 or /tmp/zpo.sock)")
	milterCmd.Flags().BoolVarP(&milterDebug, "debug", "d", false, "Enable debug logging")

	rootCmd.AddCommand(milterCmd)
}
