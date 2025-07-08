package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
)

// domain-blocker - ZPAM plugin for content-analyzer
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: domain-blocker <email-file>")
	}

	emailFile := os.Args[1]

	// Load configuration with sensible defaults
	config := loadConfiguration()

	// Parse email file
	email, err := parseEmailFile(emailFile)
	if err != nil {
		log.Printf("Warning: Error parsing email: %v", err)
		outputResult(PluginResult{
			Score:       0.0,
			Confidence:  0.1,
			Explanation: fmt.Sprintf("Email parsing failed: %v", err),
			Metadata: map[string]interface{}{
				"plugin_name": "domain-blocker",
				"version":     "1.0.0",
				"error":       "parsing_failed",
			},
		})
		return
	}

	// Analyze the domain
	result := analyzeDomain(email, config)
	outputResult(result)
}

type PluginResult struct {
	Score       float64                `json:"score"`
	Confidence  float64                `json:"confidence"`
	Explanation string                 `json:"explanation"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Email struct {
	From    string
	To      string
	Subject string
	Body    string
	Headers map[string]string
}

type DomainConfig struct {
	// High-risk domains (blocked completely)
	BlockedDomains []string `json:"blocked_domains"`
	BlockScore     float64  `json:"block_score"`

	// Suspicious domains with custom weights
	SuspiciousDomains map[string]float64 `json:"suspicious_domains"`

	// Known disposable email providers
	DisposableDomains []string `json:"disposable_domains"`
	DisposableScore   float64  `json:"disposable_score"`

	// Temporary email patterns
	TempPatterns []string `json:"temp_patterns"`
	TempScore    float64  `json:"temp_score"`

	// DNS validation settings
	CheckDNS   bool    `json:"check_dns"`
	NoDNSScore float64 `json:"no_dns_score"`
	NoMXScore  float64 `json:"no_mx_score"`

	// Pattern-based detection
	CheckPatterns      bool     `json:"check_patterns"`
	NumberScore        float64  `json:"number_score"`
	SuspiciousTLDs     []string `json:"suspicious_tlds"`
	SuspiciousTLDScore float64  `json:"suspicious_tld_score"`
}

func loadConfiguration() DomainConfig {
	return DomainConfig{
		// Blocked domains (spam score: 0.9)
		BlockedDomains: []string{
			"suspicious.com",
			"phishing-site.net",
			"malware.org",
			"fakebank.com",
			"scam-central.net",
			"fake-paypal.com",
			"notreal-bank.org",
		},
		BlockScore: 0.9,

		// Suspicious domains with custom weights
		SuspiciousDomains: map[string]float64{
			"marketing-blast.com":      0.6,
			"newsletter-spam.net":      0.5,
			"promotions-unlimited.org": 0.7,
			"deals-galore.com":         0.6,
			"clickbait-central.net":    0.8,
		},

		// Known disposable email providers
		DisposableDomains: []string{
			"10minutemail.com",
			"temp-mail.org",
			"guerrillamail.com",
			"mailinator.com",
			"throwaway-email.com",
			"yopmail.com",
			"sharklasers.com",
		},
		DisposableScore: 0.8,

		// Temporary email patterns
		TempPatterns: []string{
			"temp", "temporary", "disposable", "minute",
			"throw", "guerrilla", "mailinator", "test",
			"spam", "junk", "fake", "dummy",
		},
		TempScore: 0.7,

		// DNS validation
		CheckDNS:   true,
		NoDNSScore: 0.8,
		NoMXScore:  0.7,

		// Pattern-based detection
		CheckPatterns: true,
		NumberScore:   0.6,
		SuspiciousTLDs: []string{
			".tk", ".ml", ".ga", ".cf", ".gq",
			".bit", ".onion",
		},
		SuspiciousTLDScore: 0.7,
	}
}

func analyzeDomain(email Email, config DomainConfig) PluginResult {
	domain := extractDomain(email.From)
	var score float64 = 0.0
	var reasons []string
	var detections []string

	metadata := map[string]interface{}{
		"plugin_name": "domain-blocker",
		"version":     "1.0.0",
		"domain":      domain,
		"from_email":  email.From,
		"timestamp":   time.Now().Unix(),
		"checks":      []string{},
	}

	// 1. Check blocked domains (highest priority)
	if contains(config.BlockedDomains, domain) {
		score = config.BlockScore
		reasons = append(reasons, fmt.Sprintf("Domain '%s' is in blocklist", domain))
		detections = append(detections, "blocked_domain")
		metadata["blocked"] = true
	}

	// 2. Check suspicious domains with custom weights
	if score < config.BlockScore {
		if suspiciousScore, exists := config.SuspiciousDomains[domain]; exists {
			score = max(score, suspiciousScore)
			reasons = append(reasons, fmt.Sprintf("Domain '%s' marked as suspicious (weight: %.1f)", domain, suspiciousScore))
			detections = append(detections, "suspicious_domain")
			metadata["suspicious"] = true
		}
	}

	// 3. Check disposable email providers
	if contains(config.DisposableDomains, domain) {
		score = max(score, config.DisposableScore)
		reasons = append(reasons, fmt.Sprintf("Domain '%s' is a disposable email provider", domain))
		detections = append(detections, "disposable_email")
		metadata["disposable"] = true
	}

	// 4. Pattern-based checks
	if config.CheckPatterns {
		patternScore, patternReasons := checkDomainPatterns(domain, config)
		if patternScore > 0 {
			score = max(score, patternScore)
			reasons = append(reasons, patternReasons...)
			detections = append(detections, "pattern_match")
			metadata["pattern_detection"] = true
		}
	}

	// 5. DNS validation checks
	if config.CheckDNS {
		dnsScore, dnsReasons := performDNSChecks(domain, config)
		if dnsScore > 0 {
			score = max(score, dnsScore)
			reasons = append(reasons, dnsReasons...)
			detections = append(detections, "dns_issue")
			metadata["dns_check"] = true
		}
	}

	// 6. Default score for unknown domains
	if score == 0.0 {
		score = 0.1 // Low spam probability for unknown domains
		reasons = append(reasons, fmt.Sprintf("Domain '%s' appears legitimate", domain))
		detections = append(detections, "legitimate")
	}

	// Calculate confidence based on detection methods
	confidence := calculateConfidence(detections, domain)

	// Generate explanation
	explanation := generateExplanation(domain, score, reasons)

	// Add final metadata
	metadata["detections"] = detections
	metadata["confidence_level"] = getConfidenceLevel(confidence)
	metadata["risk_level"] = getRiskLevel(score)

	return PluginResult{
		Score:       score,
		Confidence:  confidence,
		Explanation: explanation,
		Metadata:    metadata,
	}
}

func checkDomainPatterns(domain string, config DomainConfig) (float64, []string) {
	var score float64 = 0.0
	var reasons []string
	domainLower := strings.ToLower(domain)

	// Check for temporary email patterns
	for _, pattern := range config.TempPatterns {
		if strings.Contains(domainLower, pattern) {
			score = max(score, config.TempScore)
			reasons = append(reasons, fmt.Sprintf("Domain contains temporary email pattern: '%s'", pattern))
			break
		}
	}

	// Check suspicious TLDs
	for _, tld := range config.SuspiciousTLDs {
		if strings.HasSuffix(domainLower, tld) {
			score = max(score, config.SuspiciousTLDScore)
			reasons = append(reasons, fmt.Sprintf("Domain uses suspicious TLD: '%s'", tld))
			break
		}
	}

	// Check for excessive numbers (often spam domains)
	numberPattern := regexp.MustCompile(`\d{3,}`)
	if numberPattern.MatchString(domain) {
		score = max(score, config.NumberScore)
		reasons = append(reasons, "Domain contains suspicious number patterns")
	}

	// Check for mixed case in domain (typosquatting indicator)
	mixedPattern := regexp.MustCompile(`[a-z][A-Z]|[A-Z][a-z]`)
	if mixedPattern.MatchString(domain) {
		score = max(score, 0.5)
		reasons = append(reasons, "Domain uses suspicious mixed case (typosquatting indicator)")
	}

	// Check for excessive hyphens or underscores
	if strings.Count(domain, "-") >= 3 || strings.Count(domain, "_") >= 2 {
		score = max(score, 0.4)
		reasons = append(reasons, "Domain contains excessive hyphens/underscores")
	}

	return score, reasons
}

func performDNSChecks(domain string, config DomainConfig) (float64, []string) {
	var score float64 = 0.0
	var reasons []string

	// Check if domain resolves
	_, err := net.LookupHost(domain)
	if err != nil {
		score = config.NoDNSScore
		reasons = append(reasons, fmt.Sprintf("Domain '%s' does not resolve (DNS lookup failed)", domain))
		return score, reasons
	}

	// Check MX records (email servers)
	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		score = max(score, config.NoMXScore)
		reasons = append(reasons, fmt.Sprintf("Domain '%s' has no MX records (cannot receive email)", domain))
	}

	// Check for suspicious MX patterns
	if len(mxRecords) > 0 {
		for _, mx := range mxRecords {
			mxHost := strings.ToLower(mx.Host)
			if strings.Contains(mxHost, "spam") || strings.Contains(mxHost, "bulk") {
				score = max(score, 0.6)
				reasons = append(reasons, fmt.Sprintf("Suspicious MX record: %s", mxHost))
				break
			}
		}
	}

	return score, reasons
}

func extractDomain(email string) string {
	// Handle email addresses like "Name <email@domain.com>"
	if angleIndex := strings.Index(email, "<"); angleIndex >= 0 {
		email = email[angleIndex+1:]
		if closeIndex := strings.Index(email, ">"); closeIndex >= 0 {
			email = email[:closeIndex]
		}
	}

	// Extract domain part
	parts := strings.Split(strings.TrimSpace(email), "@")
	if len(parts) != 2 {
		return email // Return original if not valid email format
	}

	return strings.ToLower(parts[1])
}

func parseEmailFile(filename string) (Email, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Email{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	email := Email{Headers: make(map[string]string)}
	scanner := bufio.NewScanner(file)
	inHeaders := true

	for scanner.Scan() {
		line := scanner.Text()

		if inHeaders {
			if strings.TrimSpace(line) == "" {
				inHeaders = false
				continue
			}

			// Parse headers
			if colonIndex := strings.Index(line, ": "); colonIndex > 0 {
				key := strings.TrimSpace(line[:colonIndex])
				value := strings.TrimSpace(line[colonIndex+2:])
				email.Headers[key] = value

				// Extract common headers
				switch strings.ToLower(key) {
				case "from":
					email.From = value
				case "to":
					email.To = value
				case "subject":
					email.Subject = value
				}
			}
		} else {
			// Body content
			email.Body += line + "\n"
		}
	}

	if err := scanner.Err(); err != nil {
		return email, fmt.Errorf("error reading file: %w", err)
	}

	// Validate we have minimum required fields
	if email.From == "" {
		return email, fmt.Errorf("no From header found in email")
	}

	return email, nil
}

func calculateConfidence(detections []string, domain string) float64 {
	baseConfidence := 0.7

	// Higher confidence with multiple detection methods
	numDetections := len(detections)
	switch {
	case numDetections >= 4:
		return 0.95
	case numDetections >= 3:
		return 0.90
	case numDetections >= 2:
		return 0.85
	case numDetections >= 1:
		return 0.80
	default:
		return baseConfidence
	}
}

func generateExplanation(domain string, score float64, reasons []string) string {
	if len(reasons) == 0 {
		return fmt.Sprintf("Domain '%s' appears legitimate (score: %.2f)", domain, score)
	}

	if len(reasons) == 1 {
		return fmt.Sprintf("Domain '%s' flagged (score: %.2f): %s", domain, score, reasons[0])
	}

	return fmt.Sprintf("Domain '%s' flagged (score: %.2f): %s", domain, score, strings.Join(reasons, "; "))
}

func getConfidenceLevel(confidence float64) string {
	switch {
	case confidence >= 0.9:
		return "very_high"
	case confidence >= 0.8:
		return "high"
	case confidence >= 0.7:
		return "medium"
	case confidence >= 0.6:
		return "low"
	default:
		return "very_low"
	}
}

func getRiskLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "high_risk"
	case score >= 0.6:
		return "medium_risk"
	case score >= 0.4:
		return "low_risk"
	default:
		return "minimal_risk"
	}
}

func contains(slice []string, item string) bool {
	itemLower := strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == itemLower {
			return true
		}
	}
	return false
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func outputResult(result PluginResult) {
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling result: %v", err)
	}
	fmt.Println(string(jsonResult))
}
