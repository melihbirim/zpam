package headers

import (
	"strings"
	"testing"
	"time"
)

func TestNewValidator(t *testing.T) {
	// Test with nil config (should use defaults)
	validator := NewValidator(nil)
	if validator == nil {
		t.Fatal("NewValidator returned nil")
	}
	
	// Test with custom config
	config := &Config{
		EnableSPF:             true,
		EnableDKIM:            false,
		EnableDMARC:           true,
		DNSTimeout:            2 * time.Second,
		MaxHopCount:           10,
		SuspiciousServerScore: 80,
		CacheSize:             500,
		CacheTTL:              30 * time.Minute,
	}
	
	validator = NewValidator(config)
	if validator == nil {
		t.Fatal("NewValidator returned nil with custom config")
	}
	
	if validator.config.DNSTimeout != 2*time.Second {
		t.Errorf("Expected DNS timeout 2s, got %v", validator.config.DNSTimeout)
	}
}

func TestValidateHeaders(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test headers with basic email information
	headers := map[string]string{
		"From":       "test@example.com",
		"To":         "recipient@domain.com",
		"Subject":    "Test Email",
		"Date":       "Mon, 01 Jan 2024 12:00:00 +0000",
		"Message-ID": "<test123@example.com>",
		"Return-Path": "test@example.com",
		"Received":   "from mail.example.com (mail.example.com [192.168.1.1]) by mx.domain.com",
	}
	
	result := validator.ValidateHeaders(headers)
	
	// Check that result is not nil
	if result == nil {
		t.Fatal("ValidateHeaders returned nil")
	}
	
	// Check that validation was performed
	if result.ValidatedAt.IsZero() {
		t.Error("ValidatedAt should be set")
	}
	
	if result.Duration == 0 {
		t.Error("Duration should be greater than 0")
	}
	
	// Check that SPF, DKIM, DMARC results are present
	if result.SPF.Result == "" {
		t.Error("SPF result should not be empty")
	}
	
	// Check that scores are calculated
	if result.AuthScore < 0 || result.AuthScore > 100 {
		t.Errorf("AuthScore should be 0-100, got %f", result.AuthScore)
	}
	
	if result.SuspiciScore < 0 || result.SuspiciScore > 100 {
		t.Errorf("SuspiciScore should be 0-100, got %f", result.SuspiciScore)
	}
}

func TestExtractDomain(t *testing.T) {
	validator := NewValidator(nil)
	
	testCases := []struct {
		email    string
		expected string
	}{
		{"test@example.com", "example.com"},
		{"user@DOMAIN.COM", "domain.com"},
		{"Name <user@example.org>", "example.org"},
		{"<test@domain.net>", "domain.net"},
		{"invalid-email", ""},
		{"", ""},
		{"test@", ""},
		{"@domain.com", ""},
	}
	
	for _, tc := range testCases {
		result := validator.extractDomain(tc.email)
		if result != tc.expected {
			t.Errorf("extractDomain(%q) = %q, expected %q", tc.email, result, tc.expected)
		}
	}
}

func TestSPFValidation(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test SPF validation with mock domain
	result := validator.validateSPF("example.com", "192.168.1.1")
	
	// Should have some result (even if DNS lookup fails)
	if result.Result == "" {
		t.Error("SPF result should not be empty")
	}
	
	// Valid results should be one of the standard SPF results
	validResults := []string{"pass", "fail", "softfail", "neutral", "none", "temperror", "permerror"}
	found := false
	for _, valid := range validResults {
		if result.Result == valid {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("SPF result %q is not a valid SPF result", result.Result)
	}
}

func TestDKIMValidation(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test without DKIM signature
	headers := map[string]string{
		"From": "test@example.com",
	}
	
	result := validator.validateDKIM(headers)
	if result.Valid {
		t.Error("DKIM should not be valid without signature")
	}
	if result.Explanation == "" {
		t.Error("DKIM explanation should not be empty")
	}
	
	// Test with DKIM signature
	headers["DKIM-Signature"] = "v=1; a=rsa-sha256; d=example.com; s=default; h=from:to:subject; bh=hash; b=signature"
	
	result = validator.validateDKIM(headers)
	if !result.Valid {
		t.Error("DKIM should be valid with proper signature format")
	}
	if len(result.Domains) == 0 {
		t.Error("DKIM domains should be extracted")
	}
}

func TestDMARCValidation(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test DMARC validation with mock SPF and DKIM results
	spfResult := SPFResult{Valid: true, Result: "pass"}
	dkimResult := DKIMResult{Valid: true}
	
	result := validator.validateDMARC("example.com", spfResult, dkimResult)
	
	// Should have some result (even if DNS lookup fails)
	if result.Explanation == "" {
		t.Error("DMARC explanation should not be empty")
	}
}

func TestRoutingAnalysis(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test with suspicious routing
	received := []string{
		"from suspicious.server.com (suspicious.server.com [192.168.1.1])",
		"from dynamic.pool.isp.com (dynamic.pool.isp.com [10.0.0.1])",
		"from mail.example.com (mail.example.com [203.0.113.1])",
	}
	
	result := validator.analyzeRouting(received)
	
	if result.HopCount != 3 {
		t.Errorf("Expected hop count 3, got %d", result.HopCount)
	}
	
	if len(result.SuspiciousHops) == 0 {
		t.Error("Should detect suspicious hops")
	}
	
	if len(result.OpenRelays) == 0 {
		t.Error("Should detect open relay patterns")
	}
}

func TestAnomalyDetection(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test with anomalous headers
	headers := map[string]string{
		"From":        "test@example.com",
		"Return-Path": "different@another.com",
		"Date":        "invalid-date-format",
		"Message-ID":  "invalid-message-id",
		// Missing Subject header
	}
	
	fromDomain := "example.com"
	returnPathDomain := "another.com"
	
	anomalies := validator.detectAnomalies(headers, fromDomain, returnPathDomain)
	
	if len(anomalies) == 0 {
		t.Error("Should detect anomalies")
	}
	
	// Check for specific anomalies
	found := false
	for _, anomaly := range anomalies {
		if strings.Contains(anomaly, "Domain mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should detect domain mismatch")
	}
}

func TestScoreCalculation(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test with good authentication
	goodResult := &ValidationResult{
		SPF:   SPFResult{Valid: true, Result: "pass"},
		DKIM:  DKIMResult{Valid: true},
		DMARC: DMARCResult{Valid: true},
		Routing: RoutingResult{
			HopCount:         3,
			SuspiciousHops:   []string{},
			OpenRelays:       []string{},
			ReverseDNSIssues: []string{},
		},
		Anomalies: []string{},
	}
	
	authScore := validator.calculateAuthScore(goodResult)
	suspiciousScore := validator.calculateSuspiciousScore(goodResult)
	
	if authScore < 80 {
		t.Errorf("Good authentication should have high auth score, got %f", authScore)
	}
	
	if suspiciousScore > 20 {
		t.Errorf("Good authentication should have low suspicious score, got %f", suspiciousScore)
	}
	
	// Test with bad authentication
	badResult := &ValidationResult{
		SPF:   SPFResult{Valid: false, Result: "fail"},
		DKIM:  DKIMResult{Valid: false},
		DMARC: DMARCResult{Valid: false},
		Routing: RoutingResult{
			HopCount:         15,
			SuspiciousHops:   []string{"suspicious server"},
			OpenRelays:       []string{"open relay"},
			ReverseDNSIssues: []string{"no reverse DNS"},
		},
		Anomalies: []string{"missing header", "invalid format"},
	}
	
	authScore = validator.calculateAuthScore(badResult)
	suspiciousScore = validator.calculateSuspiciousScore(badResult)
	
	if authScore > 50 {
		t.Errorf("Bad authentication should have low auth score, got %f", authScore)
	}
	
	if suspiciousScore < 50 {
		t.Errorf("Bad authentication should have high suspicious score, got %f", suspiciousScore)
	}
}

func TestHelperFunctions(t *testing.T) {
	validator := NewValidator(nil)
	
	// Test IP in CIDR
	testCases := []struct {
		ip       string
		cidr     string
		expected bool
	}{
		{"192.168.1.1", "192.168.1.0/24", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"192.168.1.1", "10.0.0.0/8", false},
		{"invalid-ip", "192.168.1.0/24", false},
		{"192.168.1.1", "invalid-cidr", false},
	}
	
	for _, tc := range testCases {
		result := validator.ipInCIDR(tc.ip, tc.cidr)
		if result != tc.expected {
			t.Errorf("ipInCIDR(%q, %q) = %v, expected %v", tc.ip, tc.cidr, result, tc.expected)
		}
	}
	
	// Test Message-ID validation
	validMessageIDs := []string{
		"<test@example.com>",
		"<12345.abcde@domain.org>",
	}
	
	for _, msgID := range validMessageIDs {
		if !validator.isValidMessageID(msgID) {
			t.Errorf("isValidMessageID(%q) should return true", msgID)
		}
	}
	
	invalidMessageIDs := []string{
		"test@example.com",
		"<invalid>",
		"<test@>",
		"",
	}
	
	for _, msgID := range invalidMessageIDs {
		if validator.isValidMessageID(msgID) {
			t.Errorf("isValidMessageID(%q) should return false", msgID)
		}
	}
}

func TestExtractDKIMParam(t *testing.T) {
	validator := NewValidator(nil)
	
	dkimHeader := "v=1; a=rsa-sha256; d=example.com; s=default; h=from:to:subject; bh=hash; b=signature"
	
	testCases := []struct {
		param    string
		expected string
	}{
		{"v", "1"},
		{"a", "rsa-sha256"},
		{"d", "example.com"},
		{"s", "default"},
		{"nonexistent", ""},
	}
	
	for _, tc := range testCases {
		result := validator.extractDKIMParam(dkimHeader, tc.param)
		if result != tc.expected {
			t.Errorf("extractDKIMParam(%q) = %q, expected %q", tc.param, result, tc.expected)
		}
	}
}

func TestPerformance(t *testing.T) {
	// Use fast config for performance test (disable DNS lookups)
	config := &Config{
		EnableSPF:             false, // Disable to avoid DNS lookups
		EnableDKIM:            true,
		EnableDMARC:           false, // Disable to avoid DNS lookups
		DNSTimeout:            100 * time.Millisecond,
		MaxHopCount:           15,
		SuspiciousServerScore: 75,
		CacheSize:             1000,
		CacheTTL:              1 * time.Hour,
	}
	validator := NewValidator(config)
	
	// Test validation performance
	headers := map[string]string{
		"From":       "test@example.com",
		"To":         "recipient@domain.com",
		"Subject":    "Test Email",
		"Date":       "Mon, 01 Jan 2024 12:00:00 +0000",
		"Message-ID": "<test123@example.com>",
		"Return-Path": "test@example.com",
		"Received":   "from mail.example.com (mail.example.com [192.168.1.1]) by mx.domain.com",
	}
	
	start := time.Now()
	result := validator.ValidateHeaders(headers)
	elapsed := time.Since(start)
	
	if result == nil {
		t.Fatal("ValidateHeaders returned nil")
	}
	
	// Headers validation should be fast (under 50ms without DNS lookups)
	if elapsed > 50*time.Millisecond {
		t.Errorf("Headers validation took too long: %v", elapsed)
	}
	
	t.Logf("Headers validation took: %v", elapsed)
} 