package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

// RspamdPlugin integrates with Rspamd for advanced spam detection
type RspamdPlugin struct {
	config     *PluginConfig
	enabled    bool
	baseURL    string
	timeout    time.Duration
	maxSize    int64
	httpClient *http.Client
	password   string // Optional password for authentication
}

// RspamdResponse represents the JSON response from Rspamd
type RspamdResponse struct {
	IsSkipped     bool                    `json:"is_skipped"`
	Score         float64                 `json:"score"`
	RequiredScore float64                 `json:"required_score"`
	Action        string                  `json:"action"`
	Symbols       map[string]RspamdSymbol `json:"symbols"`
	Messages      map[string]string       `json:"messages"`
	MessageID     string                  `json:"message-id"`
	TimeReal      float64                 `json:"time_real"`
	URLs          []string                `json:"urls"`
}

// RspamdSymbol represents a triggered rule/symbol in Rspamd
type RspamdSymbol struct {
	Score       float64            `json:"score"`
	MetricScore map[string]float64 `json:"metric_score"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Options     []string           `json:"options"`
}

// NewRspamdPlugin creates a new Rspamd plugin
func NewRspamdPlugin() *RspamdPlugin {
	return &RspamdPlugin{
		enabled: false,
		baseURL: "http://localhost:11334",
		timeout: 10 * time.Second,
		maxSize: 10 * 1024 * 1024, // 10MB max
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the plugin name
func (r *RspamdPlugin) Name() string {
	return "rspamd"
}

// Version returns the plugin version
func (r *RspamdPlugin) Version() string {
	return "1.0.0"
}

// Description returns plugin description
func (r *RspamdPlugin) Description() string {
	return "Rspamd integration plugin for modern spam and malware detection"
}

// Initialize sets up the plugin with configuration
func (r *RspamdPlugin) Initialize(config *PluginConfig) error {
	r.config = config
	r.enabled = config.Enabled

	if !r.enabled {
		return nil
	}

	// Parse plugin-specific settings
	if settings := config.Settings; settings != nil {
		if baseURL, ok := settings["base_url"].(string); ok {
			r.baseURL = baseURL
		}
		if timeout, ok := settings["timeout"].(string); ok {
			if d, err := time.ParseDuration(timeout); err == nil {
				r.timeout = d
			}
		}
		if maxSize, ok := settings["max_size"].(int64); ok {
			r.maxSize = maxSize
		}
		if password, ok := settings["password"].(string); ok {
			r.password = password
		}
	}

	// Override timeout from config
	if config.Timeout > 0 {
		r.timeout = config.Timeout
	}

	// Update HTTP client timeout
	r.httpClient.Timeout = r.timeout

	// Verify Rspamd is available
	return r.checkRspamdAvailable()
}

// IsHealthy checks if Rspamd is ready
func (r *RspamdPlugin) IsHealthy(ctx context.Context) error {
	if !r.enabled {
		return fmt.Errorf("rspamd plugin not enabled")
	}

	// Test ping endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+"/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rspamd not healthy: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rspamd returned status %d", resp.StatusCode)
	}

	return nil
}

// Cleanup releases resources
func (r *RspamdPlugin) Cleanup() error {
	// No persistent resources to clean up
	return nil
}

// AnalyzeContent implements ContentAnalyzer interface
func (r *RspamdPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	if !r.enabled {
		return &PluginResult{
			Name:        r.Name(),
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("plugin not enabled"),
		}, nil
	}

	// Format email for Rspamd
	emailContent, err := r.formatEmailForRspamd(email)
	if err != nil {
		return r.errorResult(start, fmt.Errorf("failed to format email: %v", err))
	}

	// Check size limit
	if int64(len(emailContent)) > r.maxSize {
		return r.errorResult(start, fmt.Errorf("email too large (%d bytes, max %d)", len(emailContent), r.maxSize))
	}

	// Send to Rspamd for analysis
	rspamdResp, err := r.analyzeWithRspamd(ctx, emailContent)
	if err != nil {
		return r.errorResult(start, fmt.Errorf("rspamd analysis failed: %v", err))
	}

	// Convert Rspamd response to plugin result
	result := r.convertRspamdResponse(rspamdResp)
	result.Name = r.Name()
	result.ProcessTime = time.Since(start)

	return result, nil
}

// checkRspamdAvailable verifies Rspamd is accessible
func (r *RspamdPlugin) checkRspamdAvailable() error {
	req, err := http.NewRequest("GET", r.baseURL+"/ping", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("rspamd not available at %s: %v", r.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rspamd ping failed with status %d", resp.StatusCode)
	}

	return nil
}

// formatEmailForRspamd formats email for Rspamd analysis
func (r *RspamdPlugin) formatEmailForRspamd(email *email.Email) ([]byte, error) {
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

	// Add message ID if missing
	if _, exists := email.Headers["Message-ID"]; !exists {
		content.WriteString(fmt.Sprintf("Message-ID: <%d@zpo.local>\n", time.Now().Unix()))
	}

	// Empty line separating headers from body
	content.WriteString("\n")

	// Add body
	content.WriteString(email.Body)

	return []byte(content.String()), nil
}

// analyzeWithRspamd sends email to Rspamd for analysis
func (r *RspamdPlugin) analyzeWithRspamd(ctx context.Context, emailContent []byte) (*RspamdResponse, error) {
	// Create request to check endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/checkv2", bytes.NewReader(emailContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "message/rfc822")

	// Add authentication if password is set
	if r.password != "" {
		req.Header.Set("Password", r.password)
	}

	// Send request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rspamd returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var rspamdResp RspamdResponse
	if err := json.Unmarshal(body, &rspamdResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &rspamdResp, nil
}

// convertRspamdResponse converts Rspamd response to PluginResult
func (r *RspamdPlugin) convertRspamdResponse(resp *RspamdResponse) *PluginResult {
	result := &PluginResult{
		Score:      resp.Score,
		Confidence: 0.8, // Rspamd is generally reliable
		Details:    make(map[string]any),
		Metadata:   make(map[string]string),
		Rules:      []string{},
	}

	// Add basic details
	result.Details["action"] = resp.Action
	result.Details["required_score"] = resp.RequiredScore
	result.Details["is_spam"] = resp.Action == "reject" || resp.Action == "add header"
	result.Details["is_skipped"] = resp.IsSkipped
	result.Details["processing_time"] = resp.TimeReal

	// Extract triggered symbols/rules
	var rules []string
	var totalSymbolScore float64
	symbolCount := 0

	for symbolName, symbol := range resp.Symbols {
		if symbol.Score != 0 {
			rules = append(rules, fmt.Sprintf("%s (%.1f)", symbolName, symbol.Score))
			totalSymbolScore += symbol.Score
			symbolCount++
		}
	}

	result.Rules = rules

	// Calculate confidence based on various factors
	confidence := 0.5 // Base confidence

	// Increase confidence based on number of rules triggered
	if symbolCount > 0 {
		confidence += float64(symbolCount) * 0.05
		if confidence > 0.9 {
			confidence = 0.9
		}
	}

	// Increase confidence based on score distance from threshold
	if resp.RequiredScore > 0 && resp.Score > 0 {
		scoreRatio := resp.Score / resp.RequiredScore
		if scoreRatio > 1.0 {
			confidence += (scoreRatio - 1.0) * 0.2
		}
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	result.Confidence = confidence

	// Add metadata
	result.Metadata["engine"] = "rspamd"
	result.Metadata["action"] = resp.Action
	result.Metadata["rules_count"] = fmt.Sprintf("%d", len(rules))
	result.Metadata["message_id"] = resp.MessageID

	// Add URLs if found
	if len(resp.URLs) > 0 {
		result.Details["urls"] = resp.URLs
		result.Metadata["url_count"] = fmt.Sprintf("%d", len(resp.URLs))
	}

	return result
}

// errorResult creates an error result
func (r *RspamdPlugin) errorResult(start time.Time, err error) (*PluginResult, error) {
	return &PluginResult{
		Name:        r.Name(),
		Score:       0,
		Confidence:  0,
		ProcessTime: time.Since(start),
		Error:       err,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
	}, nil
}

// GetEngineStats implements ExternalEngine interface
func (r *RspamdPlugin) GetEngineStats(ctx context.Context) (map[string]any, error) {
	if !r.enabled {
		return nil, fmt.Errorf("plugin not enabled")
	}

	stats := map[string]any{
		"name":     "Rspamd",
		"enabled":  r.enabled,
		"base_url": r.baseURL,
		"timeout":  r.timeout.String(),
		"max_size": r.maxSize,
	}

	// Try to get Rspamd stats
	if rspamdStats, err := r.getRspamdStats(ctx); err == nil {
		stats["rspamd_stats"] = rspamdStats
	}

	return stats, nil
}

// getRspamdStats retrieves Rspamd statistics
func (r *RspamdPlugin) getRspamdStats(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+"/stat", nil)
	if err != nil {
		return nil, err
	}

	if r.password != "" {
		req.Header.Set("Password", r.password)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var stats map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// Analyze implements ExternalEngine interface (alias for AnalyzeContent)
func (r *RspamdPlugin) Analyze(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return r.AnalyzeContent(ctx, email)
}
