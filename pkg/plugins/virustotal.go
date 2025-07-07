package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

// VirusTotalPlugin integrates with VirusTotal API for threat intelligence
type VirusTotalPlugin struct {
	config   *PluginConfig
	enabled  bool
	apiKey   string
	baseURL  string
	client   *http.Client
	stats    *VTStats
	urlRegex *regexp.Regexp
}

// VTStats tracks VirusTotal plugin statistics
type VTStats struct {
	URLsChecked     int64 `json:"urls_checked"`
	HashesChecked   int64 `json:"hashes_checked"`
	MaliciousFound  int64 `json:"malicious_found"`
	SuspiciousFound int64 `json:"suspicious_found"`
	APICallsTotal   int64 `json:"api_calls_total"`
	APICallsFailed  int64 `json:"api_calls_failed"`
	RateLimitHits   int64 `json:"rate_limit_hits"`
}

// VTResponse represents VirusTotal API response structure
type VTResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			LastAnalysisStats struct {
				Harmless   int `json:"harmless"`
				Malicious  int `json:"malicious"`
				Suspicious int `json:"suspicious"`
				Undetected int `json:"undetected"`
				Timeout    int `json:"timeout"`
			} `json:"last_analysis_stats"`
			LastAnalysisResults map[string]struct {
				Category string `json:"category"`
				Result   string `json:"result"`
			} `json:"last_analysis_results"`
			Reputation int    `json:"reputation"`
			URL        string `json:"url,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// VTURLSubmitResponse represents URL submission response
type VTURLSubmitResponse struct {
	Data struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"data"`
}

// NewVirusTotalPlugin creates a new VirusTotal plugin
func NewVirusTotalPlugin() *VirusTotalPlugin {
	// Compile URL regex for extraction
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)

	return &VirusTotalPlugin{
		enabled:  false,
		baseURL:  "https://www.virustotal.com/api/v3",
		client:   &http.Client{Timeout: 30 * time.Second},
		stats:    &VTStats{},
		urlRegex: urlRegex,
	}
}

// Name returns the plugin name
func (vt *VirusTotalPlugin) Name() string {
	return "virustotal"
}

// Version returns the plugin version
func (vt *VirusTotalPlugin) Version() string {
	return "1.0.0"
}

// Description returns plugin description
func (vt *VirusTotalPlugin) Description() string {
	return "VirusTotal integration for URL and file reputation checking"
}

// Initialize sets up the plugin with configuration
func (vt *VirusTotalPlugin) Initialize(config *PluginConfig) error {
	vt.config = config
	vt.enabled = config.Enabled

	if !vt.enabled {
		return nil
	}

	// Extract API key from settings
	if settings := config.Settings; settings != nil {
		if apiKey, ok := settings["api_key"].(string); ok && apiKey != "" {
			vt.apiKey = apiKey
		} else {
			return fmt.Errorf("VirusTotal API key is required")
		}

		// Configure timeout if specified
		if timeoutMs, ok := settings["timeout"].(float64); ok && timeoutMs > 0 {
			vt.client.Timeout = time.Duration(timeoutMs) * time.Millisecond
		}

		// Configure custom base URL if specified
		if baseURL, ok := settings["base_url"].(string); ok && baseURL != "" {
			vt.baseURL = baseURL
		}
	} else {
		return fmt.Errorf("VirusTotal settings are required")
	}

	return nil
}

// IsHealthy checks if the plugin is ready
func (vt *VirusTotalPlugin) IsHealthy(ctx context.Context) error {
	if !vt.enabled {
		return fmt.Errorf("VirusTotal plugin not enabled")
	}

	if vt.apiKey == "" {
		return fmt.Errorf("VirusTotal API key not configured")
	}

	// Test API connectivity with a simple request
	req, err := http.NewRequestWithContext(ctx, "GET", vt.baseURL+"/users/"+vt.apiKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %v", err)
	}

	req.Header.Set("X-Apikey", vt.apiKey)

	resp, err := vt.client.Do(req)
	if err != nil {
		return fmt.Errorf("VirusTotal API unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("VirusTotal API key invalid")
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("VirusTotal API error: %d", resp.StatusCode)
	}

	return nil
}

// Cleanup releases resources
func (vt *VirusTotalPlugin) Cleanup() error {
	// Reset stats
	vt.stats = &VTStats{}
	return nil
}

// CheckReputation implements ReputationChecker interface
func (vt *VirusTotalPlugin) CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	result := &PluginResult{
		Name:        vt.Name(),
		Score:       0,
		Confidence:  0.8, // VirusTotal is generally reliable
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
		Rules:       []string{},
		ProcessTime: 0,
	}

	if !vt.enabled {
		result.Error = fmt.Errorf("plugin not enabled")
		result.ProcessTime = time.Since(start)
		return result, nil
	}

	var totalScore float64
	var threats []string
	var checkedItems int

	// Extract and check URLs from email
	urls := vt.extractURLs(email.Body + " " + email.Subject)
	for i, urlStr := range urls {
		// Limit URL checks to prevent excessive API usage
		if i >= 5 {
			break
		}

		score, threat, err := vt.checkURL(ctx, urlStr)
		if err != nil {
			vt.stats.APICallsFailed++
			continue
		}

		totalScore += score
		checkedItems++
		vt.stats.URLsChecked++

		if threat != "" {
			threats = append(threats, fmt.Sprintf("URL: %s (%s)", urlStr, threat))
		}
	}

	// Check attachment types for suspicious patterns
	for _, attachment := range email.Attachments {
		// Score based on suspicious file types and names
		score := vt.scoreAttachmentBySuspicion(attachment)
		if score > 0 {
			totalScore += score
			checkedItems++
			vt.stats.HashesChecked++
			threats = append(threats, fmt.Sprintf("Suspicious attachment: %s (%s)",
				attachment.Filename, attachment.ContentType))
		}
	}

	// Calculate final score and confidence
	if checkedItems > 0 {
		result.Score = totalScore
		result.Confidence = 0.9 // High confidence when we have data

		if len(threats) > 0 {
			result.Rules = threats
			if totalScore >= 10 {
				vt.stats.MaliciousFound++
			} else if totalScore >= 5 {
				vt.stats.SuspiciousFound++
			}
		}
	}

	// Add metadata
	result.Metadata["urls_checked"] = fmt.Sprintf("%d", len(urls))
	result.Metadata["files_checked"] = fmt.Sprintf("%d", len(email.Attachments))
	result.Metadata["threats_found"] = fmt.Sprintf("%d", len(threats))

	// Add details
	result.Details["checked_items"] = checkedItems
	result.Details["total_score"] = totalScore
	if len(threats) > 0 {
		result.Details["threats"] = threats
	}

	result.ProcessTime = time.Since(start)
	return result, nil
}

// extractURLs extracts URLs from email content
func (vt *VirusTotalPlugin) extractURLs(content string) []string {
	matches := vt.urlRegex.FindAllString(content, -1)

	// Deduplicate URLs
	seen := make(map[string]bool)
	var unique []string

	for _, match := range matches {
		// Clean up URL (remove trailing punctuation)
		cleaned := strings.TrimRight(match, ".,;!?)")
		if !seen[cleaned] {
			seen[cleaned] = true
			unique = append(unique, cleaned)
		}
	}

	return unique
}

// scoreAttachmentBySuspicion scores attachments based on file type and name patterns
func (vt *VirusTotalPlugin) scoreAttachmentBySuspicion(attachment email.Attachment) float64 {
	filename := strings.ToLower(attachment.Filename)
	contentType := strings.ToLower(attachment.ContentType)

	var score float64

	// Highly suspicious executable file types
	executableTypes := []string{
		".exe", ".scr", ".bat", ".cmd", ".com", ".pif", ".vbs", ".js", ".jar",
		".app", ".deb", ".rpm", ".dmg", ".pkg", ".msi", ".ps1",
	}

	for _, ext := range executableTypes {
		if strings.HasSuffix(filename, ext) {
			score += 12.0
			break
		}
	}

	// Suspicious archive types that could contain malware
	archiveTypes := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2"}
	for _, ext := range archiveTypes {
		if strings.HasSuffix(filename, ext) {
			score += 3.0
			break
		}
	}

	// Double extensions (common in malware)
	if strings.Count(filename, ".") > 1 {
		score += 4.0
	}

	// Suspicious content types
	if strings.Contains(contentType, "application/octet-stream") ||
		strings.Contains(contentType, "application/x-msdownload") ||
		strings.Contains(contentType, "application/x-executable") {
		score += 5.0
	}

	// Large file size could indicate payload
	if attachment.Size > 10*1024*1024 { // 10MB
		score += 2.0
	}

	// Suspicious filename patterns
	suspiciousPatterns := []string{
		"invoice", "receipt", "urgent", "payment", "refund",
		"document", "scan", "fax", "photo", "img",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(filename, pattern) {
			score += 1.5
			break
		}
	}

	return score
}

// checkURL checks URL reputation with VirusTotal
func (vt *VirusTotalPlugin) checkURL(ctx context.Context, urlStr string) (float64, string, error) {
	// Encode URL for VirusTotal API
	urlID := vt.encodeURL(urlStr)

	req, err := http.NewRequestWithContext(ctx, "GET",
		vt.baseURL+"/urls/"+urlID, nil)
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("X-Apikey", vt.apiKey)

	resp, err := vt.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	vt.stats.APICallsTotal++

	if resp.StatusCode == 429 {
		vt.stats.RateLimitHits++
		return 0, "", fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode == 404 {
		// URL not found, submit for analysis
		return vt.submitURL(ctx, urlStr)
	}

	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var vtResp VTResponse
	if err := json.NewDecoder(resp.Body).Decode(&vtResp); err != nil {
		return 0, "", err
	}

	// Calculate threat score based on detections
	stats := vtResp.Data.Attributes.LastAnalysisStats
	totalEngines := stats.Harmless + stats.Malicious + stats.Suspicious + stats.Undetected

	if totalEngines == 0 {
		return 0, "", nil
	}

	maliciousRatio := float64(stats.Malicious) / float64(totalEngines)
	suspiciousRatio := float64(stats.Suspicious) / float64(totalEngines)

	var score float64
	var threatType string

	if maliciousRatio > 0.1 { // More than 10% of engines detect as malicious
		score = 15.0 + (maliciousRatio * 10.0) // 15-25 points
		threatType = "malicious"
	} else if suspiciousRatio > 0.2 { // More than 20% detect as suspicious
		score = 5.0 + (suspiciousRatio * 10.0) // 5-15 points
		threatType = "suspicious"
	} else if stats.Malicious > 0 {
		score = 3.0 // Low score for minimal detections
		threatType = "low_risk"
	}

	return score, threatType, nil
}

// submitURL submits URL for analysis if not found
func (vt *VirusTotalPlugin) submitURL(ctx context.Context, urlStr string) (float64, string, error) {
	data := url.Values{}
	data.Set("url", urlStr)

	req, err := http.NewRequestWithContext(ctx, "POST",
		vt.baseURL+"/urls", strings.NewReader(data.Encode()))
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("X-Apikey", vt.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := vt.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	vt.stats.APICallsTotal++

	if resp.StatusCode == 429 {
		vt.stats.RateLimitHits++
		return 0, "", fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("submit API error: %d", resp.StatusCode)
	}

	// URL submitted successfully, return neutral score
	return 0, "", nil
}

// checkFileHash checks file hash reputation
func (vt *VirusTotalPlugin) checkFileHash(ctx context.Context, hash string) (float64, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		vt.baseURL+"/files/"+hash, nil)
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("X-Apikey", vt.apiKey)

	resp, err := vt.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	vt.stats.APICallsTotal++

	if resp.StatusCode == 429 {
		vt.stats.RateLimitHits++
		return 0, "", fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode == 404 {
		// File not found in VirusTotal database
		return 0, "", nil
	}

	if resp.StatusCode != 200 {
		return 0, "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var vtResp VTResponse
	if err := json.NewDecoder(resp.Body).Decode(&vtResp); err != nil {
		return 0, "", err
	}

	// Calculate threat score based on detections
	stats := vtResp.Data.Attributes.LastAnalysisStats
	totalEngines := stats.Harmless + stats.Malicious + stats.Suspicious + stats.Undetected

	if totalEngines == 0 {
		return 0, "", nil
	}

	maliciousRatio := float64(stats.Malicious) / float64(totalEngines)
	suspiciousRatio := float64(stats.Suspicious) / float64(totalEngines)

	var score float64
	var threatType string

	if maliciousRatio > 0.05 { // More than 5% detect as malicious (stricter for files)
		score = 20.0 + (maliciousRatio * 15.0) // 20-35 points
		threatType = "malware"
	} else if suspiciousRatio > 0.15 { // More than 15% detect as suspicious
		score = 8.0 + (suspiciousRatio * 12.0) // 8-20 points
		threatType = "suspicious_file"
	} else if stats.Malicious > 0 {
		score = 4.0 // Low score for minimal detections
		threatType = "potentially_unwanted"
	}

	return score, threatType, nil
}

// calculateSHA256 calculates SHA256 hash of content
func (vt *VirusTotalPlugin) calculateSHA256(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

// encodeURL encodes URL for VirusTotal API (base64url without padding)
func (vt *VirusTotalPlugin) encodeURL(urlStr string) string {
	// VirusTotal uses URL-safe base64 encoding without padding
	encoded := base64.URLEncoding.EncodeToString([]byte(urlStr))
	return strings.TrimRight(encoded, "=")
}

// GetStats returns plugin statistics
func (vt *VirusTotalPlugin) GetStats() *VTStats {
	return vt.stats
}

// GetEngineStats implements ExternalEngine interface
func (vt *VirusTotalPlugin) GetEngineStats(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"urls_checked":     vt.stats.URLsChecked,
		"hashes_checked":   vt.stats.HashesChecked,
		"malicious_found":  vt.stats.MaliciousFound,
		"suspicious_found": vt.stats.SuspiciousFound,
		"api_calls_total":  vt.stats.APICallsTotal,
		"api_calls_failed": vt.stats.APICallsFailed,
		"rate_limit_hits":  vt.stats.RateLimitHits,
		"api_key_present":  vt.apiKey != "",
	}, nil
}

// Analyze implements ExternalEngine interface (alias for CheckReputation)
func (vt *VirusTotalPlugin) Analyze(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return vt.CheckReputation(ctx, email)
}
