package headers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"net"

	"github.com/zpo/spam-filter/pkg/dns"
)

// ValidationResult contains email header validation results
type ValidationResult struct {
	// Authentication results
	SPF   SPFResult   `json:"spf"`
	DKIM  DKIMResult  `json:"dkim"`
	DMARC DMARCResult `json:"dmarc"`
	
	// Routing analysis
	Routing RoutingResult `json:"routing"`
	
	// Header anomalies
	Anomalies []string `json:"anomalies"`
	
	// Overall scores
	AuthScore    float64 `json:"auth_score"`    // 0-100 (higher = better auth)
	SuspiciScore float64 `json:"suspici_score"` // 0-100 (higher = more suspicious)
	
	// Validation metadata
	ValidatedAt time.Time `json:"validated_at"`
	Duration    time.Duration `json:"duration"`
	
	// Cache performance statistics
	CacheStats CachePerformance `json:"cache_stats"`
}

// CachePerformance contains cache performance metrics
type CachePerformance struct {
	HitRate       float64 `json:"hit_rate"`         // Overall DNS cache hit rate (%)
	TotalEntries  int64   `json:"total_entries"`    // Total cached entries
	TotalHits     int64   `json:"total_hits"`       // Total cache hits
	TotalMisses   int64   `json:"total_misses"`     // Total cache misses
	TotalErrors   int64   `json:"total_errors"`     // Total DNS errors
	TotalEvictions int64  `json:"total_evictions"`  // Total cache evictions
}

// SPFResult contains SPF validation results
type SPFResult struct {
	Valid       bool     `json:"valid"`
	Record      string   `json:"record"`
	Result      string   `json:"result"`      // pass, fail, softfail, neutral, none, temperror, permerror
	Explanation string   `json:"explanation"`
	IPMatches   []string `json:"ip_matches"`
}

// DKIMResult contains DKIM validation results
type DKIMResult struct {
	Valid         bool     `json:"valid"`
	Signatures    []string `json:"signatures"`
	Domains       []string `json:"domains"`
	Selectors     []string `json:"selectors"`
	Algorithms    []string `json:"algorithms"`
	Explanation   string   `json:"explanation"`
}

// DMARCResult contains DMARC validation results
type DMARCResult struct {
	Valid       bool   `json:"valid"`
	Policy      string `json:"policy"`      // none, quarantine, reject
	Alignment   string `json:"alignment"`   // relaxed, strict
	Percentage  int    `json:"percentage"`
	Explanation string `json:"explanation"`
}

// RoutingResult contains email routing analysis
type RoutingResult struct {
	HopCount         int      `json:"hop_count"`
	SuspiciousHops   []string `json:"suspicious_hops"`
	OpenRelays       []string `json:"open_relays"`
	GeoAnomalies     []string `json:"geo_anomalies"`
	TimingAnomalies  []string `json:"timing_anomalies"`
	ReverseDNSIssues []string `json:"reverse_dns_issues"`
}

// DNSClient interface for DNS operations
type DNSClient interface {
	LookupTXT(domain string) ([]string, error)
	LookupA(domain string) ([]net.IP, error)
	LookupMX(domain string) ([]*net.MX, error)
	GetSPFRecord(domain string) (string, error)
	GetDMARCRecord(domain string) (string, error)
	CheckIPInA(domain, ip string) (bool, error)
	CheckIPInMX(domain, ip string) (bool, error)
	ValidateReverseDNS(ip string) (bool, error)
	GetStats() dns.Stats
	ClearCache()
	ResetStats()
	HitRate() float64
}

// Validator handles email header validation
type Validator struct {
	config    *Config
	dnsClient DNSClient
}

// Config contains validation configuration
type Config struct {
	// Enable/disable validations
	EnableSPF   bool `json:"enable_spf"`
	EnableDKIM  bool `json:"enable_dkim"`
	EnableDMARC bool `json:"enable_dmarc"`
	
	// Timeouts
	DNSTimeout time.Duration `json:"dns_timeout"`
	
	// Thresholds
	MaxHopCount           int `json:"max_hop_count"`
	SuspiciousServerScore int `json:"suspicious_server_score"`
	
	// Known suspicious patterns
	SuspiciousServers []string `json:"suspicious_servers"`
	OpenRelayPatterns []string `json:"open_relay_patterns"`
	
	// Cache settings
	CacheSize int           `json:"cache_size"`
	CacheTTL  time.Duration `json:"cache_ttl"`
}

// DefaultConfig returns default header validation configuration
func DefaultConfig() *Config {
	return &Config{
		EnableSPF:             true,
		EnableDKIM:            true,
		EnableDMARC:           true,
		DNSTimeout:            5 * time.Second,
		MaxHopCount:           15,
		SuspiciousServerScore: 75,
		SuspiciousServers: []string{
			"suspicious", "spam", "bulk", "mass", "marketing",
			"promo", "offer", "deal", "free", "win",
		},
		OpenRelayPatterns: []string{
			"unknown", "dynamic", "dhcp", "dial", "cable",
			"dsl", "adsl", "pool", "client", "user",
		},
		CacheSize: 1000,
		CacheTTL:  1 * time.Hour,
	}
}

// NewValidator creates a new header validator
func NewValidator(config *Config) *Validator {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create DNS client with caching
	dnsConfig := dns.Config{
		Timeout:       config.DNSTimeout,
		CacheSize:     config.CacheSize,
		CacheTTL:      config.CacheTTL,
		EnableCaching: true,
	}
	
	return &Validator{
		config:    config,
		dnsClient: dns.NewClient(dnsConfig),
	}
}

// NewTestValidator creates a header validator using the test DNS client
func NewTestValidator(config *Config, testServer *dns.TestServer) *Validator {
	if config == nil {
		config = DefaultConfig()
	}
	
	// Create DNS client with test server
	dnsConfig := dns.Config{
		Timeout:       config.DNSTimeout,
		CacheSize:     config.CacheSize,
		CacheTTL:      config.CacheTTL,
		EnableCaching: true,
	}
	
	testClient := dns.NewTestClient(dnsConfig, testServer)
	
	return &Validator{
		config:    config,
		dnsClient: testClient,
	}
}

// ValidateHeaders validates all email headers
func (v *Validator) ValidateHeaders(headers map[string]string) *ValidationResult {
	start := time.Now()
	
	result := &ValidationResult{
		ValidatedAt: start,
		Anomalies:   make([]string, 0),
	}
	
	// Extract key information
	from := headers["From"]
	returnPath := headers["Return-Path"]
	received := v.extractReceivedHeaders(headers)
	
	// Domain extraction
	fromDomain := v.extractDomain(from)
	returnPathDomain := v.extractDomain(returnPath)
	
	// SPF Validation
	if v.config.EnableSPF && fromDomain != "" {
		result.SPF = v.validateSPF(fromDomain, v.extractClientIP(received))
	}
	
	// DKIM Validation
	if v.config.EnableDKIM {
		result.DKIM = v.validateDKIM(headers)
	}
	
	// DMARC Validation
	if v.config.EnableDMARC && fromDomain != "" {
		result.DMARC = v.validateDMARC(fromDomain, result.SPF, result.DKIM)
	}
	
	// Routing Analysis
	result.Routing = v.analyzeRouting(received)
	
	// Header Anomalies
	result.Anomalies = v.detectAnomalies(headers, fromDomain, returnPathDomain)
	
	// Calculate scores
	result.AuthScore = v.calculateAuthScore(result)
	result.SuspiciScore = v.calculateSuspiciousScore(result)
	
	// Collect cache statistics
	result.CacheStats = v.collectCacheStats()
	
	result.Duration = time.Since(start)
	return result
}

// validateSPF validates SPF record for domain
func (v *Validator) validateSPF(domain, clientIP string) SPFResult {
	result := SPFResult{
		IPMatches: make([]string, 0),
	}
	
	// Get SPF record using DNS client
	spfRecord, err := v.dnsClient.GetSPFRecord(domain)
	if err != nil {
		result.Result = "none"
		result.Explanation = "No SPF record found"
		return result
	}
	
	result.Record = spfRecord
	result.Result = v.evaluateSPF(spfRecord, clientIP, domain)
	result.Valid = (result.Result == "pass")
	
	return result
}

// evaluateSPF evaluates SPF record against client IP
func (v *Validator) evaluateSPF(record, clientIP, domain string) string {
	if clientIP == "" {
		return "neutral"
	}
	
	// Parse SPF mechanisms
	mechanisms := strings.Fields(record)
	
	for _, mechanism := range mechanisms[1:] { // Skip "v=spf1"
		if strings.HasPrefix(mechanism, "ip4:") {
			cidr := strings.TrimPrefix(mechanism, "ip4:")
			if v.ipInCIDR(clientIP, cidr) {
				return "pass"
			}
		} else if strings.HasPrefix(mechanism, "ip6:") {
			// IPv6 support (simplified)
			continue
		} else if strings.HasPrefix(mechanism, "include:") {
			includeDomain := strings.TrimPrefix(mechanism, "include:")
			includeResult := v.validateSPF(includeDomain, clientIP)
			if includeResult.Result == "pass" {
				return "pass"
			}
		} else if mechanism == "a" {
			// Check if client IP matches domain A record
			if v.checkARecord(domain, clientIP) {
				return "pass"
			}
		} else if mechanism == "mx" {
			// Check if client IP matches domain MX record
			if v.checkMXRecord(domain, clientIP) {
				return "pass"
			}
		} else if strings.HasPrefix(mechanism, "-") {
			// Hard fail
			return "fail"
		} else if strings.HasPrefix(mechanism, "~") {
			// Soft fail
			return "softfail"
		}
	}
	
	// Default is usually neutral or soft fail
	return "neutral"
}

// validateDKIM validates DKIM signatures
func (v *Validator) validateDKIM(headers map[string]string) DKIMResult {
	result := DKIMResult{
		Signatures: make([]string, 0),
		Domains:    make([]string, 0),
		Selectors:  make([]string, 0),
		Algorithms: make([]string, 0),
	}
	
	// Look for DKIM-Signature header
	dkimHeader := headers["DKIM-Signature"]
	if dkimHeader == "" {
		result.Explanation = "No DKIM signature found"
		return result
	}
	
	result.Signatures = append(result.Signatures, dkimHeader)
	
	// Parse DKIM signature components
	domain := v.extractDKIMParam(dkimHeader, "d")
	selector := v.extractDKIMParam(dkimHeader, "s")
	algorithm := v.extractDKIMParam(dkimHeader, "a")
	
	if domain != "" {
		result.Domains = append(result.Domains, domain)
	}
	if selector != "" {
		result.Selectors = append(result.Selectors, selector)
	}
	if algorithm != "" {
		result.Algorithms = append(result.Algorithms, algorithm)
	}
	
	// Simplified validation (in production, would verify signature)
	result.Valid = (domain != "" && selector != "" && algorithm != "")
	
	if result.Valid {
		result.Explanation = "DKIM signature appears valid"
	} else {
		result.Explanation = "DKIM signature malformed"
	}
	
	return result
}

// validateDMARC validates DMARC policy
func (v *Validator) validateDMARC(domain string, spf SPFResult, dkim DKIMResult) DMARCResult {
	result := DMARCResult{}
	
	// Get DMARC record using DNS client
	dmarcRecord, err := v.dnsClient.GetDMARCRecord(domain)
	if err != nil {
		result.Explanation = "No DMARC record found"
		return result
	}
	
	// Parse DMARC policy
	result.Policy = v.extractDMARCParam(dmarcRecord, "p")
	result.Alignment = v.extractDMARCParam(dmarcRecord, "adkim")
	if result.Alignment == "" {
		result.Alignment = "relaxed" // Default
	}
	
	// Parse percentage
	if pct := v.extractDMARCParam(dmarcRecord, "pct"); pct != "" {
		if percentage, err := strconv.Atoi(pct); err == nil {
			result.Percentage = percentage
		}
	} else {
		result.Percentage = 100 // Default
	}
	
	// Check alignment (simplified)
	spfAligned := (spf.Result == "pass")
	dkimAligned := dkim.Valid
	
	result.Valid = (spfAligned || dkimAligned)
	
	if result.Valid {
		result.Explanation = "DMARC alignment satisfied"
	} else {
		result.Explanation = "DMARC alignment failed"
	}
	
	return result
}

// analyzeRouting analyzes email routing path
func (v *Validator) analyzeRouting(received []string) RoutingResult {
	result := RoutingResult{
		HopCount:         len(received),
		SuspiciousHops:   make([]string, 0),
		OpenRelays:       make([]string, 0),
		GeoAnomalies:     make([]string, 0),
		TimingAnomalies:  make([]string, 0),
		ReverseDNSIssues: make([]string, 0),
	}
	
	// Analyze each hop
	for i, hop := range received {
		// Check for suspicious servers
		for _, suspicious := range v.config.SuspiciousServers {
			if strings.Contains(strings.ToLower(hop), suspicious) {
				result.SuspiciousHops = append(result.SuspiciousHops, 
					fmt.Sprintf("Hop %d: suspicious server pattern '%s'", i+1, suspicious))
			}
		}
		
		// Check for open relay patterns
		for _, pattern := range v.config.OpenRelayPatterns {
			if strings.Contains(strings.ToLower(hop), pattern) {
				result.OpenRelays = append(result.OpenRelays, 
					fmt.Sprintf("Hop %d: open relay pattern '%s'", i+1, pattern))
			}
		}
		
		// Extract and validate IPs
		ips := v.extractIPs(hop)
		for _, ip := range ips {
			// Check reverse DNS
			if !v.validateReverseDNS(ip) {
				result.ReverseDNSIssues = append(result.ReverseDNSIssues, 
					fmt.Sprintf("Hop %d: no reverse DNS for %s", i+1, ip))
			}
		}
	}
	
	return result
}

// detectAnomalies detects header anomalies
func (v *Validator) detectAnomalies(headers map[string]string, fromDomain, returnPathDomain string) []string {
	anomalies := make([]string, 0)
	
	// Check From/Return-Path domain mismatch
	if fromDomain != "" && returnPathDomain != "" && fromDomain != returnPathDomain {
		anomalies = append(anomalies, 
			fmt.Sprintf("Domain mismatch: From=%s, Return-Path=%s", fromDomain, returnPathDomain))
	}
	
	// Check for missing critical headers
	criticalHeaders := []string{"From", "Date", "Message-ID"}
	for _, header := range criticalHeaders {
		if headers[header] == "" {
			anomalies = append(anomalies, fmt.Sprintf("Missing header: %s", header))
		}
	}
	
	// Check for suspicious header values
	if messageID := headers["Message-ID"]; messageID != "" {
		if !v.isValidMessageID(messageID) {
			anomalies = append(anomalies, "Invalid Message-ID format")
		}
	}
	
	// Check for duplicate headers (simplified - would need full email parsing)
	
	// Check Date header
	if date := headers["Date"]; date != "" {
		if !v.isValidDate(date) {
			anomalies = append(anomalies, "Invalid Date header format")
		} else {
			// Check if date is too far in past/future
			if parsedDate, err := time.Parse(time.RFC1123Z, date); err == nil {
				now := time.Now()
				if now.Sub(parsedDate) > 7*24*time.Hour {
					anomalies = append(anomalies, "Date too far in past")
				} else if parsedDate.Sub(now) > 24*time.Hour {
					anomalies = append(anomalies, "Date in future")
				}
			}
		}
	}
	
	return anomalies
}

// Helper functions

func (v *Validator) extractDomain(email string) string {
	if email == "" {
		return ""
	}
	
	// Extract from angle brackets if present
	if strings.Contains(email, "<") && strings.Contains(email, ">") {
		start := strings.Index(email, "<") + 1
		end := strings.Index(email, ">")
		if start < end {
			email = email[start:end]
		}
	}
	
	// Extract domain part
	parts := strings.Split(email, "@")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return strings.ToLower(parts[1])
	}
	
	return ""
}

func (v *Validator) extractReceivedHeaders(headers map[string]string) []string {
	var received []string
	
	// In a real implementation, would need to handle multiple Received headers
	if r := headers["Received"]; r != "" {
		received = append(received, r)
	}
	
	return received
}

func (v *Validator) extractClientIP(received []string) string {
	if len(received) == 0 {
		return ""
	}
	
	// Extract IP from first Received header (simplified)
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	matches := ipRegex.FindStringSubmatch(received[0])
	if len(matches) > 0 {
		return matches[0]
	}
	
	return ""
}

func (v *Validator) extractDKIMParam(header, param string) string {
	// Look for param=value in DKIM header
	pattern := regexp.MustCompile(param + `=([^;]+)`)
	matches := pattern.FindStringSubmatch(header)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (v *Validator) extractDMARCParam(record, param string) string {
	// Look for param=value in DMARC record
	pattern := regexp.MustCompile(param + `=([^;]+)`)
	matches := pattern.FindStringSubmatch(record)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (v *Validator) extractIPs(text string) []string {
	ipRegex := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	return ipRegex.FindAllString(text, -1)
}

func (v *Validator) ipInCIDR(ip, cidr string) bool {
	// Simple IP in CIDR check
	if !strings.Contains(cidr, "/") {
		return ip == cidr
	}
	
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	
	testIP := net.ParseIP(ip)
	if testIP == nil {
		return false
	}
	
	return ipNet.Contains(testIP)
}

func (v *Validator) checkARecord(domain, ip string) bool {
	match, err := v.dnsClient.CheckIPInA(domain, ip)
	if err != nil {
		return false
	}
	return match
}

func (v *Validator) checkMXRecord(domain, ip string) bool {
	match, err := v.dnsClient.CheckIPInMX(domain, ip)
	if err != nil {
		return false
	}
	return match
}

func (v *Validator) validateReverseDNS(ip string) bool {
	valid, err := v.dnsClient.ValidateReverseDNS(ip)
	if err != nil {
		return false
	}
	return valid
}

func (v *Validator) isValidMessageID(messageID string) bool {
	// Basic Message-ID format validation
	if !strings.HasPrefix(messageID, "<") || !strings.HasSuffix(messageID, ">") {
		return false
	}
	
	// Extract content between angle brackets
	content := messageID[1 : len(messageID)-1]
	
	// Must contain @ and have content before and after @
	parts := strings.Split(content, "@")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func (v *Validator) isValidDate(date string) bool {
	// Try common date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, date); err == nil {
			return true
		}
	}
	
	return false
}

func (v *Validator) calculateAuthScore(result *ValidationResult) float64 {
	score := 50.0 // Base score
	
	// SPF contribution (30 points)
	switch result.SPF.Result {
	case "pass":
		score += 30
	case "fail":
		score -= 20
	case "softfail":
		score -= 10
	}
	
	// DKIM contribution (30 points)
	if result.DKIM.Valid {
		score += 30
	} else {
		score -= 15
	}
	
	// DMARC contribution (20 points)
	if result.DMARC.Valid {
		score += 20
	} else {
		score -= 10
	}
	
	// Penalties for anomalies
	score -= float64(len(result.Anomalies)) * 5
	
	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

func (v *Validator) calculateSuspiciousScore(result *ValidationResult) float64 {
	score := 0.0
	
	// Authentication failures
	if result.SPF.Result == "fail" {
		score += 30
	} else if result.SPF.Result == "softfail" {
		score += 15
	}
	
	if !result.DKIM.Valid {
		score += 20
	}
	
	if !result.DMARC.Valid {
		score += 25
	}
	
	// Routing issues
	score += float64(len(result.Routing.SuspiciousHops)) * 10
	score += float64(len(result.Routing.OpenRelays)) * 15
	score += float64(len(result.Routing.ReverseDNSIssues)) * 5
	
	// Header anomalies
	score += float64(len(result.Anomalies)) * 8
	
	// Excessive hops
	if result.Routing.HopCount > v.config.MaxHopCount {
		score += 20
	}
	
	// Clamp to 0-100
	if score > 100 {
		score = 100
	}
	
	return score
}

// collectCacheStats collects cache performance statistics
func (v *Validator) collectCacheStats() CachePerformance {
	stats := v.dnsClient.GetStats()
	
	return CachePerformance{
		HitRate:        v.dnsClient.HitRate(),
		TotalEntries:   stats.Entries,
		TotalHits:      stats.Hits,
		TotalMisses:    stats.Misses,
		TotalErrors:    stats.Errors,
		TotalEvictions: stats.Evictions,
	}
}

// GetCacheStats returns current cache performance statistics
func (v *Validator) GetCacheStats() CachePerformance {
	return v.collectCacheStats()
}

// ClearCaches clears all DNS caches
func (v *Validator) ClearCaches() {
	v.dnsClient.ClearCache()
}

// ResetCacheStats resets all cache statistics
func (v *Validator) ResetCacheStats() {
	v.dnsClient.ResetStats()
} 