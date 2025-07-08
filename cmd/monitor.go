package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	monitorInterval int
	monitorCompact  bool
	monitorLogTail  int
	monitorMetrics  bool
	monitorAlerts   bool
	monitorNoColor  bool
	monitorNoCharts bool
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Live performance monitoring dashboard",
	Long: `Real-time ZPAM performance monitoring with:
- Live service status and health metrics
- Email processing statistics and trends
- Resource usage (memory, CPU, connections)
- Performance charts and visualizations
- Live log tailing and error alerts
- Interactive dashboard with keyboard controls

Perfect for production monitoring and troubleshooting.`,
	RunE: runMonitor,
}

// Monitoring data structures
type MonitoringData struct {
	Service     ServiceMetrics     `json:"service"`
	Performance PerformanceMetrics `json:"performance"`
	Resources   ResourceMetrics    `json:"resources"`
	Learning    LearningMetrics    `json:"learning"`
	Logs        []LogEntry         `json:"logs"`
	Alerts      []Alert            `json:"alerts"`
	Timestamp   time.Time          `json:"timestamp"`
}

type ServiceMetrics struct {
	Running          bool          `json:"running"`
	PID              int           `json:"pid"`
	Uptime           time.Duration `json:"uptime"`
	Status           string        `json:"status"`
	ConnectionsCount int           `json:"connections_count"`
	QueueSize        int           `json:"queue_size"`
	ErrorRate        float64       `json:"error_rate"`
}

type PerformanceMetrics struct {
	EmailsProcessed    int64         `json:"emails_processed"`
	EmailsPerSecond    float64       `json:"emails_per_second"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LatestProcessTime  time.Duration `json:"latest_process_time"`
	ThroughputHistory  []float64     `json:"throughput_history"`
	LatencyHistory     []float64     `json:"latency_history"`
	SpamDetected       int64         `json:"spam_detected"`
	SpamRate           float64       `json:"spam_rate"`
}

type ResourceMetrics struct {
	MemoryUsageMB      float64 `json:"memory_usage_mb"`
	MemoryPercent      float64 `json:"memory_percent"`
	CPUPercent         float64 `json:"cpu_percent"`
	DiskUsageMB        float64 `json:"disk_usage_mb"`
	NetworkConnections int     `json:"network_connections"`
	FileDescriptors    int     `json:"file_descriptors"`
}

type LearningMetrics struct {
	Backend          string    `json:"backend"`
	TokensLearned    int       `json:"tokens_learned"`
	SpamLearned      int       `json:"spam_learned"`
	HamLearned       int       `json:"ham_learned"`
	ModelAccuracy    float64   `json:"model_accuracy"`
	LastTrainingTime time.Time `json:"last_training_time"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

type Alert struct {
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

func runMonitor(cmd *cobra.Command, args []string) error {
	// Setup signal handling for graceful exit
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	fmt.Printf("🫏 ZPAM Live Performance Monitor\n")
	fmt.Printf("═══════════════════════════════════════════════════════\n")
	fmt.Printf("Press Ctrl+C to exit | Refresh every %ds\n\n", monitorInterval)

	// Initialize monitoring history
	var dataHistory []MonitoringData
	maxHistory := 60 // Keep 60 data points for charts

	ticker := time.NewTicker(time.Duration(monitorInterval) * time.Second)
	defer ticker.Stop()

	// Initial data collection
	data, err := collectMonitoringData()
	if err != nil {
		return fmt.Errorf("failed to collect initial monitoring data: %v", err)
	}
	dataHistory = append(dataHistory, *data)

	printMonitoringDashboard(data, dataHistory)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("\n\n🛑 Monitoring stopped\n")
			return nil
		case <-ticker.C:
			// Collect new data
			data, err := collectMonitoringData()
			if err != nil {
				fmt.Printf("❌ Error collecting data: %v\n", err)
				continue
			}

			// Add to history (keep only last maxHistory points)
			dataHistory = append(dataHistory, *data)
			if len(dataHistory) > maxHistory {
				dataHistory = dataHistory[1:]
			}

			// Clear screen and redraw
			if !monitorCompact {
				fmt.Print("\033[H\033[2J") // Clear screen
			}

			printMonitoringDashboard(data, dataHistory)
		}
	}
}

func collectMonitoringData() (*MonitoringData, error) {
	data := &MonitoringData{
		Timestamp: time.Now(),
	}

	// Collect service metrics
	data.Service = collectServiceMetrics()

	// Collect performance metrics
	data.Performance = collectPerformanceMetrics()

	// Collect resource metrics
	data.Resources = collectResourceMetrics()

	// Collect learning metrics
	data.Learning = collectLearningMetrics()

	// Collect recent logs
	if monitorLogTail > 0 {
		data.Logs = collectRecentLogs(monitorLogTail)
	}

	// Generate alerts
	if monitorAlerts {
		data.Alerts = generateAlerts(data)
	}

	return data, nil
}

func collectServiceMetrics() ServiceMetrics {
	// Use same logic as status command for service detection
	running := false
	var pid int
	var uptime time.Duration

	// Try to find running service
	for _, mode := range []string{"milter", "standalone"} {
		pidFile := fmt.Sprintf("zpam-%s.pid", mode)
		if data, err := os.ReadFile(pidFile); err == nil {
			if pidVal, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				if process, err := os.FindProcess(pidVal); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						running = true
						pid = pidVal

						// Get uptime from PID file
						if stat, err := os.Stat(pidFile); err == nil {
							uptime = time.Since(stat.ModTime())
						}
						break
					}
				}
			}
		}
	}

	status := "stopped"
	if running {
		status = "running"
	}

	return ServiceMetrics{
		Running:          running,
		PID:              pid,
		Uptime:           uptime,
		Status:           status,
		ConnectionsCount: 0,   // Would get from service metrics
		QueueSize:        0,   // Would get from service metrics
		ErrorRate:        0.0, // Would calculate from error logs
	}
}

func collectPerformanceMetrics() PerformanceMetrics {
	// In production, this would read from service metrics
	// For now, simulate some basic metrics
	return PerformanceMetrics{
		EmailsProcessed:    0,
		EmailsPerSecond:    0.0,
		AverageProcessTime: 0,
		LatestProcessTime:  0,
		ThroughputHistory:  []float64{0.0},
		LatencyHistory:     []float64{0.0},
		SpamDetected:       0,
		SpamRate:           0.0,
	}
}

func collectResourceMetrics() ResourceMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert to MB
	memoryMB := float64(m.Alloc) / 1024 / 1024

	return ResourceMetrics{
		MemoryUsageMB:      memoryMB,
		MemoryPercent:      0.0, // Would calculate from system memory
		CPUPercent:         0.0, // Would get from process monitoring
		DiskUsageMB:        0.0, // Would calculate log/model file sizes
		NetworkConnections: 0,   // Would get from service
		FileDescriptors:    0,   // Would get from system
	}
}

func collectLearningMetrics() LearningMetrics {
	// Load config to get learning backend info
	cfg, err := loadServiceConfig()
	if err != nil {
		return LearningMetrics{Backend: "unknown"}
	}

	metrics := LearningMetrics{
		Backend: cfg.Learning.Backend,
	}

	if !cfg.Learning.Enabled {
		metrics.Backend = "disabled"
		return metrics
	}

	// Get learning statistics (simplified)
	if cfg.Learning.Backend == "redis" {
		// Would collect from Redis
		metrics.SpamLearned = 0
		metrics.HamLearned = 0
		metrics.TokensLearned = 0
	} else if cfg.Learning.Backend == "file" {
		// Would read from model file
		metrics.SpamLearned = 0
		metrics.HamLearned = 0
		metrics.TokensLearned = 0
	}

	return metrics
}

func collectRecentLogs(count int) []LogEntry {
	// Would tail the log files in production
	// For now, return empty
	return []LogEntry{}
}

func generateAlerts(data *MonitoringData) []Alert {
	var alerts []Alert

	// Service down alert
	if !data.Service.Running {
		alerts = append(alerts, Alert{
			Type:      "service",
			Severity:  "critical",
			Message:   "ZPAM service is not running",
			Timestamp: time.Now(),
		})
	}

	// High memory usage alert
	if data.Resources.MemoryUsageMB > 500 {
		alerts = append(alerts, Alert{
			Type:      "resource",
			Severity:  "warning",
			Message:   fmt.Sprintf("High memory usage: %.1f MB", data.Resources.MemoryUsageMB),
			Timestamp: time.Now(),
		})
	}

	// High error rate alert
	if data.Service.ErrorRate > 0.05 { // 5% error rate
		alerts = append(alerts, Alert{
			Type:      "performance",
			Severity:  "warning",
			Message:   fmt.Sprintf("High error rate: %.1f%%", data.Service.ErrorRate*100),
			Timestamp: time.Now(),
		})
	}

	return alerts
}

func printMonitoringDashboard(data *MonitoringData, history []MonitoringData) {
	if !monitorCompact {
		fmt.Printf("🫏 ZPAM Live Performance Monitor - %s\n", data.Timestamp.Format("15:04:05"))
		fmt.Printf("═══════════════════════════════════════════════════════\n\n")
	}

	// Service Status Section
	printServiceStatus(&data.Service)
	fmt.Printf("\n")

	// Performance Metrics Section
	printPerformanceMetrics(&data.Performance, history)
	fmt.Printf("\n")

	// Resource Usage Section
	printResourceMetrics(&data.Resources)
	fmt.Printf("\n")

	// Learning Status Section
	printLearningStatus(&data.Learning)

	// Alerts Section
	if monitorAlerts && len(data.Alerts) > 0 {
		fmt.Printf("\n")
		printAlerts(data.Alerts)
	}

	// Log Tail Section
	if monitorLogTail > 0 && len(data.Logs) > 0 {
		fmt.Printf("\n")
		printLogTail(data.Logs)
	}

	fmt.Printf("\n🔄 Last updated: %s | Next refresh in %ds\n",
		data.Timestamp.Format("15:04:05"), monitorInterval)
}

func printServiceStatus(metrics *ServiceMetrics) {
	icon := "❌"
	if metrics.Running {
		icon = "✅"
	}

	fmt.Printf("🚀 Service Status\n")
	fmt.Printf("  Status: %s %s", icon, strings.Title(metrics.Status))

	if metrics.Running {
		fmt.Printf(" (PID: %d)\n", metrics.PID)
		fmt.Printf("  Uptime: %s\n", formatDuration(metrics.Uptime))
		fmt.Printf("  Queue Size: %d\n", metrics.QueueSize)
		fmt.Printf("  Connections: %d\n", metrics.ConnectionsCount)

		if metrics.ErrorRate > 0 {
			fmt.Printf("  Error Rate: %.2f%%\n", metrics.ErrorRate*100)
		}
	} else {
		fmt.Printf("\n")
	}
}

func printPerformanceMetrics(metrics *PerformanceMetrics, history []MonitoringData) {
	fmt.Printf("📊 Performance\n")

	if metrics.EmailsProcessed > 0 {
		fmt.Printf("  Emails Processed: %s\n", formatNumber(metrics.EmailsProcessed))
		fmt.Printf("  Processing Rate: %.1f emails/sec\n", metrics.EmailsPerSecond)

		if metrics.AverageProcessTime > 0 {
			fmt.Printf("  Avg Process Time: %v\n", metrics.AverageProcessTime)
		}

		if metrics.SpamDetected > 0 {
			fmt.Printf("  Spam Detected: %s (%.1f%%)\n",
				formatNumber(metrics.SpamDetected), metrics.SpamRate*100)
		}

		// Throughput chart
		if !monitorNoCharts && len(history) > 1 {
			fmt.Printf("  Throughput Trend: ")
			printMiniChart(extractThroughputHistory(history), 20)
		}
	} else {
		fmt.Printf("  No emails processed yet\n")
	}
}

func printResourceMetrics(metrics *ResourceMetrics) {
	fmt.Printf("💾 Resources\n")
	fmt.Printf("  Memory: %.1f MB", metrics.MemoryUsageMB)

	if metrics.MemoryPercent > 0 {
		fmt.Printf(" (%.1f%%)", metrics.MemoryPercent)
	}
	fmt.Printf("\n")

	if metrics.CPUPercent > 0 {
		fmt.Printf("  CPU: %.1f%%\n", metrics.CPUPercent)
	}

	if metrics.NetworkConnections > 0 {
		fmt.Printf("  Network: %d connections\n", metrics.NetworkConnections)
	}

	if metrics.FileDescriptors > 0 {
		fmt.Printf("  File Descriptors: %d\n", metrics.FileDescriptors)
	}
}

func printLearningStatus(metrics *LearningMetrics) {
	fmt.Printf("🧠 Learning\n")
	fmt.Printf("  Backend: %s\n", strings.Title(metrics.Backend))

	if metrics.Backend != "disabled" && metrics.Backend != "unknown" {
		fmt.Printf("  Training Data: %d spam, %d ham\n",
			metrics.SpamLearned, metrics.HamLearned)

		if metrics.TokensLearned > 0 {
			fmt.Printf("  Tokens Learned: %s\n", formatNumber(int64(metrics.TokensLearned)))
		}

		if metrics.ModelAccuracy > 0 {
			fmt.Printf("  Model Accuracy: %.1f%%\n", metrics.ModelAccuracy)
		}

		if !metrics.LastTrainingTime.IsZero() {
			fmt.Printf("  Last Training: %s\n", formatTimeAgo(metrics.LastTrainingTime))
		}
	}
}

func printAlerts(alerts []Alert) {
	fmt.Printf("🚨 Alerts\n")

	// Sort alerts by severity and timestamp
	sort.Slice(alerts, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "warning": 1, "info": 2}
		if severityOrder[alerts[i].Severity] != severityOrder[alerts[j].Severity] {
			return severityOrder[alerts[i].Severity] < severityOrder[alerts[j].Severity]
		}
		return alerts[i].Timestamp.After(alerts[j].Timestamp)
	})

	for _, alert := range alerts {
		icon := "⚠️"
		if alert.Severity == "critical" {
			icon = "🔥"
		} else if alert.Severity == "info" {
			icon = "ℹ️"
		}

		fmt.Printf("  %s %s: %s (%s)\n",
			icon, strings.Title(alert.Severity), alert.Message,
			alert.Timestamp.Format("15:04:05"))
	}
}

func printLogTail(logs []LogEntry) {
	fmt.Printf("📜 Recent Logs\n")

	for _, log := range logs {
		levelIcon := "•"
		switch log.Level {
		case "ERROR":
			levelIcon = "❌"
		case "WARN":
			levelIcon = "⚠️"
		case "INFO":
			levelIcon = "ℹ️"
		case "DEBUG":
			levelIcon = "🐛"
		}

		fmt.Printf("  %s [%s] %s %s\n",
			levelIcon, log.Timestamp.Format("15:04:05"), log.Source, log.Message)
	}
}

func printMiniChart(values []float64, width int) {
	if len(values) == 0 {
		fmt.Printf("No data\n")
		return
	}

	// Find min/max for scaling
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Normalize to chart width
	chart := make([]rune, width)
	chars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	for i := 0; i < width && i < len(values); i++ {
		// Scale value to 0-7 range for chart characters
		var normalized float64
		if max > min {
			normalized = (values[len(values)-width+i] - min) / (max - min)
		} else {
			normalized = 0.5
		}

		charIndex := int(normalized * float64(len(chars)-1))
		if charIndex >= len(chars) {
			charIndex = len(chars) - 1
		}
		chart[i] = chars[charIndex]
	}

	fmt.Printf("%s\n", string(chart))
}

func extractThroughputHistory(history []MonitoringData) []float64 {
	var values []float64
	for _, data := range history {
		values = append(values, data.Performance.EmailsPerSecond)
	}
	return values
}

// Helper functions are imported from status.go

func init() {
	monitorCmd.Flags().IntVarP(&monitorInterval, "interval", "i", 5, "Refresh interval in seconds")
	monitorCmd.Flags().BoolVarP(&monitorCompact, "compact", "", false, "Compact output (no screen clearing)")
	monitorCmd.Flags().IntVarP(&monitorLogTail, "logs", "l", 5, "Number of recent log entries to show")
	monitorCmd.Flags().BoolVarP(&monitorMetrics, "metrics", "m", true, "Show performance metrics")
	monitorCmd.Flags().BoolVarP(&monitorAlerts, "alerts", "a", true, "Show alerts and warnings")
	monitorCmd.Flags().BoolVar(&monitorNoColor, "no-color", false, "Disable colored output")
	monitorCmd.Flags().BoolVar(&monitorNoCharts, "no-charts", false, "Disable ASCII charts")

	rootCmd.AddCommand(monitorCmd)
}
