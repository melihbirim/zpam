package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zpam/spam-filter/pkg/email"
)

// SpamAssassinPlugin integrates with SpamAssassin for enhanced spam detection
type SpamAssassinPlugin struct {
	config     *PluginConfig
	enabled    bool
	saExec     string // Path to spamc or spamassassin executable
	timeout    time.Duration
	maxSize    int64  // Maximum email size to process (bytes)
	useSpamD   bool   // Use spamd daemon instead of standalone
	daemonHost string // Spamd daemon host
	daemonPort int    // Spamd daemon port
}

// NewSpamAssassinPlugin creates a new SpamAssassin plugin
func NewSpamAssassinPlugin() *SpamAssassinPlugin {
	return &SpamAssassinPlugin{
		enabled:    false,
		saExec:     "spamc", // Default to spamc (client)
		timeout:    10 * time.Second,
		maxSize:    10 * 1024 * 1024, // 10MB max
		useSpamD:   true,             // Prefer daemon for performance
		daemonHost: "localhost",
		daemonPort: 783,
	}
}

// Name returns the plugin name
func (sa *SpamAssassinPlugin) Name() string {
	return "spamassassin"
}

// Version returns the plugin version
func (sa *SpamAssassinPlugin) Version() string {
	return "1.0.0"
}

// Description returns plugin description
func (sa *SpamAssassinPlugin) Description() string {
	return "SpamAssassin integration plugin for comprehensive spam analysis"
}

// Initialize sets up the plugin with configuration
func (sa *SpamAssassinPlugin) Initialize(config *PluginConfig) error {
	sa.config = config
	sa.enabled = config.Enabled

	if !sa.enabled {
		return nil
	}

	// Parse plugin-specific settings
	if settings := config.Settings; settings != nil {
		if exec, ok := settings["executable"].(string); ok {
			sa.saExec = exec
		}
		if timeout, ok := settings["timeout"].(string); ok {
			if d, err := time.ParseDuration(timeout); err == nil {
				sa.timeout = d
			}
		}
		if maxSize, ok := settings["max_size"].(int64); ok {
			sa.maxSize = maxSize
		}
		if useSpamD, ok := settings["use_spamd"].(bool); ok {
			sa.useSpamD = useSpamD
		}
		if host, ok := settings["daemon_host"].(string); ok {
			sa.daemonHost = host
		}
		if port, ok := settings["daemon_port"].(int); ok {
			sa.daemonPort = port
		}
	}

	// Override timeout from config
	if config.Timeout > 0 {
		sa.timeout = config.Timeout
	}

	// Verify SpamAssassin is available
	return sa.checkSpamAssassinAvailable()
}

// IsHealthy checks if SpamAssassin is ready
func (sa *SpamAssassinPlugin) IsHealthy(ctx context.Context) error {
	if !sa.enabled {
		return fmt.Errorf("spamassassin plugin not enabled")
	}

	// Test with a simple command
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if sa.useSpamD {
		cmd = exec.CommandContext(timeoutCtx, sa.saExec, "--version")
	} else {
		cmd = exec.CommandContext(timeoutCtx, "spamassassin", "--version")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("spamassassin not healthy: %v", err)
	}

	return nil
}

// Cleanup releases resources
func (sa *SpamAssassinPlugin) Cleanup() error {
	// No persistent resources to clean up
	return nil
}

// AnalyzeContent implements ContentAnalyzer interface
func (sa *SpamAssassinPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	if !sa.enabled {
		return &PluginResult{
			Name:        sa.Name(),
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("plugin not enabled"),
		}, nil
	}

	// Create email content for SpamAssassin
	emailContent, err := sa.formatEmailForSA(email)
	if err != nil {
		return sa.errorResult(start, fmt.Errorf("failed to format email: %v", err))
	}

	// Check size limit
	if int64(len(emailContent)) > sa.maxSize {
		return sa.errorResult(start, fmt.Errorf("email too large (%d bytes, max %d)", len(emailContent), sa.maxSize))
	}

	// Run SpamAssassin analysis
	saResult, err := sa.runSpamAssassin(ctx, emailContent)
	if err != nil {
		return sa.errorResult(start, fmt.Errorf("spamassassin analysis failed: %v", err))
	}

	// Parse SpamAssassin output
	result, err := sa.parseSpamAssassinOutput(saResult)
	if err != nil {
		return sa.errorResult(start, fmt.Errorf("failed to parse output: %v", err))
	}

	result.Name = sa.Name()
	result.ProcessTime = time.Since(start)

	return result, nil
}

// checkSpamAssassinAvailable verifies SpamAssassin installation
func (sa *SpamAssassinPlugin) checkSpamAssassinAvailable() error {
	var cmd *exec.Cmd

	if sa.useSpamD {
		// Check spamc client
		cmd = exec.Command(sa.saExec, "--version")
	} else {
		// Check standalone spamassassin
		cmd = exec.Command("spamassassin", "--version")
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("spamassassin not available (tried %s): %v", sa.saExec, err)
	}

	return nil
}

// formatEmailForSA formats email for SpamAssassin input
func (sa *SpamAssassinPlugin) formatEmailForSA(email *email.Email) ([]byte, error) {
	var content strings.Builder

	// Add headers
	if email.Headers != nil {
		for key, value := range email.Headers {
			content.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	// Add essential headers if missing
	if _, exists := email.Headers["From"]; !exists && email.From != "" {
		content.WriteString(fmt.Sprintf("From: %s\n", email.From))
	}
	if _, exists := email.Headers["To"]; !exists && len(email.To) > 0 {
		content.WriteString(fmt.Sprintf("To: %s\n", strings.Join(email.To, ", ")))
	}
	if _, exists := email.Headers["Subject"]; !exists && email.Subject != "" {
		content.WriteString(fmt.Sprintf("Subject: %s\n", email.Subject))
	}

	// Empty line separating headers from body
	content.WriteString("\n")

	// Add body
	content.WriteString(email.Body)

	return []byte(content.String()), nil
}

// runSpamAssassin executes SpamAssassin on email content
func (sa *SpamAssassinPlugin) runSpamAssassin(ctx context.Context, emailContent []byte) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, sa.timeout)
	defer cancel()

	var cmd *exec.Cmd

	if sa.useSpamD {
		// Use spamc client to connect to spamd daemon
		args := []string{
			"-c", // Check only (don't modify)
			"-R", // Print full report
		}

		if sa.daemonHost != "localhost" || sa.daemonPort != 783 {
			args = append(args, "-d", fmt.Sprintf("%s:%d", sa.daemonHost, sa.daemonPort))
		}

		cmd = exec.CommandContext(timeoutCtx, sa.saExec, args...)
	} else {
		// Use standalone spamassassin
		cmd = exec.CommandContext(timeoutCtx, "spamassassin",
			"--check-only",
			"--report",
			"--no-user-config",
		)
	}

	// Set up stdin/stdout
	cmd.Stdin = strings.NewReader(string(emailContent))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// SpamAssassin returns non-zero exit code for spam, so check actual error
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("spamassassin timeout after %v", sa.timeout)
		}
		// Non-zero exit is normal for spam detection
	}

	return string(output), nil
}

// parseSpamAssassinOutput parses SpamAssassin analysis results
func (sa *SpamAssassinPlugin) parseSpamAssassinOutput(output string) (*PluginResult, error) {
	result := &PluginResult{
		Score:      0,
		Confidence: 0.5, // Default confidence
		Details:    make(map[string]any),
		Metadata:   make(map[string]string),
		Rules:      []string{},
	}

	lines := strings.Split(output, "\n")
	var rulesTriggered []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse score line: "X-Spam-Score: 15.2"
		if strings.HasPrefix(line, "X-Spam-Score:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "X-Spam-Score:"))
			if score, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				result.Score = score
			}
		}

		// Parse status line: "X-Spam-Status: Yes, score=15.2 required=5.0"
		if strings.HasPrefix(line, "X-Spam-Status:") {
			if strings.Contains(line, "Yes") {
				result.Details["is_spam"] = true
			} else {
				result.Details["is_spam"] = false
			}

			// Extract required threshold
			thresholdRegex := regexp.MustCompile(`required=([0-9.]+)`)
			if matches := thresholdRegex.FindStringSubmatch(line); len(matches) > 1 {
				if threshold, err := strconv.ParseFloat(matches[1], 64); err == nil {
					result.Details["threshold"] = threshold
				}
			}
		}

		// Parse individual rules: " 2.1 BAYES_99" or " * 2.1 BAYES_99"
		ruleRegex := regexp.MustCompile(`^\s*\*?\s*([0-9.-]+)\s+([A-Z_]+)`)
		if matches := ruleRegex.FindStringSubmatch(line); len(matches) > 2 {
			ruleName := matches[2]
			ruleScore := matches[1]
			rulesTriggered = append(rulesTriggered, fmt.Sprintf("%s (%.1s)", ruleName, ruleScore))
		}

		// Parse tests line: "X-Spam-Tests: BAYES_99,FREEMAIL_FROM,..."
		if strings.HasPrefix(line, "X-Spam-Tests:") {
			testsStr := strings.TrimSpace(strings.TrimPrefix(line, "X-Spam-Tests:"))
			if testsStr != "" {
				tests := strings.Split(testsStr, ",")
				for _, test := range tests {
					test = strings.TrimSpace(test)
					if test != "" {
						rulesTriggered = append(rulesTriggered, test)
					}
				}
			}
		}
	}

	result.Rules = rulesTriggered

	// Calculate confidence based on score and threshold
	if threshold, ok := result.Details["threshold"].(float64); ok {
		if result.Score > 0 {
			// Confidence increases with distance from threshold
			confidence := 0.5 + (result.Score/threshold)*0.4
			if confidence > 1.0 {
				confidence = 1.0
			}
			result.Confidence = confidence
		}
	}

	// Add metadata
	result.Metadata["engine"] = "spamassassin"
	result.Metadata["rules_count"] = fmt.Sprintf("%d", len(rulesTriggered))

	return result, nil
}

// errorResult creates an error result
func (sa *SpamAssassinPlugin) errorResult(start time.Time, err error) (*PluginResult, error) {
	return &PluginResult{
		Name:        sa.Name(),
		Score:       0,
		Confidence:  0,
		ProcessTime: time.Since(start),
		Error:       err,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
	}, nil
}

// GetEngineStats implements ExternalEngine interface
func (sa *SpamAssassinPlugin) GetEngineStats(ctx context.Context) (map[string]any, error) {
	if !sa.enabled {
		return nil, fmt.Errorf("plugin not enabled")
	}

	stats := map[string]any{
		"name":        "SpamAssassin",
		"enabled":     sa.enabled,
		"use_daemon":  sa.useSpamD,
		"daemon_host": sa.daemonHost,
		"daemon_port": sa.daemonPort,
		"timeout":     sa.timeout.String(),
		"max_size":    sa.maxSize,
	}

	// Try to get SpamAssassin version
	if version, err := sa.getSpamAssassinVersion(ctx); err == nil {
		stats["version"] = version
	}

	return stats, nil
}

// getSpamAssassinVersion retrieves SpamAssassin version
func (sa *SpamAssassinPlugin) getSpamAssassinVersion(ctx context.Context) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if sa.useSpamD {
		cmd = exec.CommandContext(timeoutCtx, sa.saExec, "--version")
	} else {
		cmd = exec.CommandContext(timeoutCtx, "spamassassin", "--version")
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output
	versionRegex := regexp.MustCompile(`SpamAssassin version ([0-9.]+)`)
	if matches := versionRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		return matches[1], nil
	}

	return strings.TrimSpace(string(output)), nil
}

// Analyze implements ExternalEngine interface (alias for AnalyzeContent)
func (sa *SpamAssassinPlugin) Analyze(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return sa.AnalyzeContent(ctx, email)
}
