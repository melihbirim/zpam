package dns

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TestRecord represents a DNS test record with TTL
type TestRecord struct {
	Type    string        `json:"type"`    // A, TXT, MX, PTR
	Value   interface{}   `json:"value"`   // Record value
	TTL     time.Duration `json:"ttl"`     // Time to live
	Created time.Time     `json:"created"` // When record was created
}

// TestServer provides fake DNS responses for testing
type TestServer struct {
	mu      sync.RWMutex
	records map[string]*TestRecord // domain:type -> record
	stats   TestServerStats
}

// TestServerStats tracks test server performance
type TestServerStats struct {
	TotalQueries  int64 `json:"total_queries"`
	CacheHits     int64 `json:"cache_hits"`
	RecordMisses  int64 `json:"record_misses"`
	ExpiredRecords int64 `json:"expired_records"`
}

// NewTestServer creates a new DNS test server
func NewTestServer() *TestServer {
	return &TestServer{
		records: make(map[string]*TestRecord),
		stats:   TestServerStats{},
	}
}

// AddARecord adds an A record for testing
func (ts *TestServer) AddARecord(domain string, ip string, ttl time.Duration) error {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	key := fmt.Sprintf("%s:A", domain)
	ts.records[key] = &TestRecord{
		Type:    "A",
		Value:   []net.IP{parsedIP},
		TTL:     ttl,
		Created: time.Now(),
	}
	
	return nil
}

// AddTXTRecord adds a TXT record for testing
func (ts *TestServer) AddTXTRecord(domain string, records []string, ttl time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	key := fmt.Sprintf("%s:TXT", domain)
	ts.records[key] = &TestRecord{
		Type:    "TXT",
		Value:   records,
		TTL:     ttl,
		Created: time.Now(),
	}
}

// AddMXRecord adds an MX record for testing
func (ts *TestServer) AddMXRecord(domain string, mx string, priority int, ttl time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	key := fmt.Sprintf("%s:MX", domain)
	mxRecord := &net.MX{
		Host: mx,
		Pref: uint16(priority),
	}
	
	ts.records[key] = &TestRecord{
		Type:    "MX",
		Value:   []*net.MX{mxRecord},
		TTL:     ttl,
		Created: time.Now(),
	}
}

// AddSPFRecord adds an SPF record for testing
func (ts *TestServer) AddSPFRecord(domain string, spfRecord string, ttl time.Duration) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	key := fmt.Sprintf("%s:TXT", domain)
	
	// Check if TXT record already exists
	if existing, exists := ts.records[key]; exists {
		// Add SPF to existing TXT records
		txtRecords := existing.Value.([]string)
		txtRecords = append(txtRecords, spfRecord)
		existing.Value = txtRecords
	} else {
		// Create new TXT record with SPF
		ts.records[key] = &TestRecord{
			Type:    "TXT",
			Value:   []string{spfRecord},
			TTL:     ttl,
			Created: time.Now(),
		}
	}
}

// AddDMARCRecord adds a DMARC record for testing
func (ts *TestServer) AddDMARCRecord(domain string, dmarcRecord string, ttl time.Duration) {
	dmarcDomain := "_dmarc." + domain
	ts.AddTXTRecord(dmarcDomain, []string{dmarcRecord}, ttl)
}

// LookupA simulates A record lookup
func (ts *TestServer) LookupA(domain string) ([]net.IP, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.stats.TotalQueries++
	
	key := fmt.Sprintf("%s:A", domain)
	record, exists := ts.records[key]
	
	if !exists {
		ts.stats.RecordMisses++
		return nil, fmt.Errorf("no A record found for domain %s", domain)
	}
	
	// Check if record has expired
	if time.Since(record.Created) > record.TTL {
		ts.stats.ExpiredRecords++
		delete(ts.records, key)
		return nil, fmt.Errorf("A record for domain %s has expired", domain)
	}
	
	ts.stats.CacheHits++
	return record.Value.([]net.IP), nil
}

// LookupTXT simulates TXT record lookup
func (ts *TestServer) LookupTXT(domain string) ([]string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.stats.TotalQueries++
	
	key := fmt.Sprintf("%s:TXT", domain)
	record, exists := ts.records[key]
	
	if !exists {
		ts.stats.RecordMisses++
		return nil, fmt.Errorf("no TXT record found for domain %s", domain)
	}
	
	// Check if record has expired
	if time.Since(record.Created) > record.TTL {
		ts.stats.ExpiredRecords++
		delete(ts.records, key)
		return nil, fmt.Errorf("TXT record for domain %s has expired", domain)
	}
	
	ts.stats.CacheHits++
	return record.Value.([]string), nil
}

// LookupMX simulates MX record lookup
func (ts *TestServer) LookupMX(domain string) ([]*net.MX, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	ts.stats.TotalQueries++
	
	key := fmt.Sprintf("%s:MX", domain)
	record, exists := ts.records[key]
	
	if !exists {
		ts.stats.RecordMisses++
		return nil, fmt.Errorf("no MX record found for domain %s", domain)
	}
	
	// Check if record has expired
	if time.Since(record.Created) > record.TTL {
		ts.stats.ExpiredRecords++
		delete(ts.records, key)
		return nil, fmt.Errorf("MX record for domain %s has expired", domain)
	}
	
	ts.stats.CacheHits++
	return record.Value.([]*net.MX), nil
}

// GetStats returns test server statistics
func (ts *TestServer) GetStats() TestServerStats {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.stats
}

// ResetStats resets all statistics
func (ts *TestServer) ResetStats() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.stats = TestServerStats{}
}

// CleanupExpired removes all expired records
func (ts *TestServer) CleanupExpired() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	var cleaned int
	now := time.Now()
	
	for key, record := range ts.records {
		if now.Sub(record.Created) > record.TTL {
			delete(ts.records, key)
			cleaned++
		}
	}
	
	return cleaned
}

// GetRecordCount returns the number of active records
func (ts *TestServer) GetRecordCount() int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return len(ts.records)
}

// LoadTestData populates the server with realistic test data
func (ts *TestServer) LoadTestData() {
	// Clear existing records
	ts.mu.Lock()
	ts.records = make(map[string]*TestRecord)
	ts.mu.Unlock()
	
	// Add realistic DNS records with varying TTLs
	
	// Gmail/Google domains - long TTL (stable infrastructure)
	ts.AddARecord("gmail.com", "142.250.191.5", 30*time.Minute)
	ts.AddTXTRecord("gmail.com", []string{
		"v=spf1 redirect=_spf.google.com",
		"google-site-verification=TV9-DBe4R80X4v0M4U_bd_J9cpOJM0nikft0jAgjmsQ",
	}, 30*time.Minute)
	ts.AddMXRecord("gmail.com", "gmail-smtp-in.l.google.com.", 5, 30*time.Minute)
	ts.AddDMARCRecord("gmail.com", "v=DMARC1; p=none; rua=mailto:mailauth-reports@google.com", 30*time.Minute)
	
	// Google SPF infrastructure
	ts.AddTXTRecord("_spf.google.com", []string{
		"v=spf1 include:_netblocks.google.com include:_netblocks2.google.com include:_netblocks3.google.com ~all",
	}, 30*time.Minute)
	
	// Microsoft/Outlook domains - medium TTL
	ts.AddARecord("outlook.com", "52.97.148.102", 15*time.Minute)
	ts.AddTXTRecord("outlook.com", []string{
		"v=spf1 include:spf-a.outlook.com include:spf-b.outlook.com ip4:157.55.9.128/25 include:spf.protection.outlook.com include:spf-a.hotmail.com include:_spf-ssg-b.microsoft.com include:_spf-ssg-c.microsoft.com ~all",
		"MS=ms23982398",
	}, 15*time.Minute)
	ts.AddMXRecord("outlook.com", "outlook-com.olc.protection.outlook.com.", 10, 15*time.Minute)
	ts.AddDMARCRecord("outlook.com", "v=DMARC1; p=none; pct=100; rua=mailto:d@rua.agari.com; ruf=mailto:d@ruf.agari.com; fo=1", 15*time.Minute)
	
	// Yahoo domains - medium TTL
	ts.AddARecord("yahoo.com", "74.6.143.25", 20*time.Minute)
	ts.AddTXTRecord("yahoo.com", []string{
		"v=spf1 redirect=_spf.mail.yahoo.com",
		"yahoo-verification-key=3jQFqN8U9k2Dxk2T5L8F",
	}, 20*time.Minute)
	ts.AddMXRecord("yahoo.com", "mta5.am0.yahoodns.net.", 1, 20*time.Minute)
	ts.AddDMARCRecord("yahoo.com", "v=DMARC1; p=reject; pct=100; rua=mailto:dmarc-yahoo-rua@yahoo-inc.com;", 20*time.Minute)
	
	// Test domains from our test data - short TTL (testing scenarios)
	ts.AddARecord("government.gov", "192.168.1.100", 5*time.Minute)
	ts.AddTXTRecord("government.gov", []string{
		"v=spf1 mx include:_spf.gov ~all",
		"gov-verification=abc123def456",
	}, 5*time.Minute)
	ts.AddMXRecord("government.gov", "mail.government.gov.", 10, 5*time.Minute)
	ts.AddDMARCRecord("government.gov", "v=DMARC1; p=quarantine; rua=mailto:dmarc@government.gov", 5*time.Minute)
	
	// Spam domains - very short TTL (frequently changing)
	ts.AddARecord("scam-alert.biz", "10.0.0.1", 1*time.Minute)
	ts.AddTXTRecord("scam-alert.biz", []string{
		"v=spf1 a mx ~all", // Weak SPF policy
	}, 1*time.Minute)
	ts.AddMXRecord("scam-alert.biz", "mail.scam-alert.biz.", 10, 1*time.Minute)
	// No DMARC record for spam domain (realistic)
	
	ts.AddARecord("test.org", "203.0.113.42", 2*time.Minute)
	ts.AddTXTRecord("test.org", []string{
		"v=spf1 include:_spf.test.org ~all",
		"test-domain-verification=xyz789",
	}, 2*time.Minute)
	ts.AddMXRecord("test.org", "mx1.test.org.", 5, 2*time.Minute)
	ts.AddDMARCRecord("test.org", "v=DMARC1; p=none; rua=mailto:admin@test.org", 2*time.Minute)
	
	// Phishing domains - very short TTL
	ts.AddARecord("phishing-site.net", "198.51.100.123", 30*time.Second)
	ts.AddTXTRecord("phishing-site.net", []string{
		"v=spf1 +all", // Very permissive SPF (suspicious)
	}, 30*time.Second)
	// No MX or DMARC records (typical for phishing)
	
	// Corporate domains - long TTL
	ts.AddARecord("example.com", "93.184.216.34", 1*time.Hour)
	ts.AddTXTRecord("example.com", []string{
		"v=spf1 -all", // No mail allowed (example domain)
	}, 1*time.Hour)
	ts.AddDMARCRecord("example.com", "v=DMARC1; p=reject; rua=mailto:dmarc@example.com", 1*time.Hour)
}

// TestClient creates a DNS client that uses the test server
type TestClient struct {
	*Client
	testServer *TestServer
}

// Ensure TestClient satisfies the same interface as Client
var _ interface {
	LookupTXT(domain string) ([]string, error)
	LookupA(domain string) ([]net.IP, error)
	LookupMX(domain string) ([]*net.MX, error)
	GetSPFRecord(domain string) (string, error)
	GetDMARCRecord(domain string) (string, error)
	CheckIPInA(domain, ip string) (bool, error)
	CheckIPInMX(domain, ip string) (bool, error)
	ValidateReverseDNS(ip string) (bool, error)
	GetStats() Stats
	ClearCache()
	ResetStats()
	HitRate() float64
} = (*TestClient)(nil)

// NewTestClient creates a DNS client configured to use the test server
func NewTestClient(config Config, testServer *TestServer) *TestClient {
	return &TestClient{
		Client:     NewClient(config),
		testServer: testServer,
	}
}

// LookupTXT overrides to use test server
func (tc *TestClient) LookupTXT(domain string) ([]string, error) {
	cacheKey := fmt.Sprintf("TXT:%s", domain)
	
	// Check cache first
	if tc.config.EnableCaching {
		if cached := tc.getFromCache(cacheKey); cached != nil {
			tc.mu.Lock()
			tc.stats.Hits++
			tc.mu.Unlock()
			return cached.([]string), nil
		}
	}
	
	// Use test server instead of real DNS
	records, err := tc.testServer.LookupTXT(domain)
	
	tc.mu.Lock()
	if err != nil {
		tc.stats.Errors++
		tc.mu.Unlock()
		return nil, err
	}
	tc.stats.Misses++
	tc.mu.Unlock()
	
	// Cache the result
	if tc.config.EnableCaching {
		tc.setInCache(cacheKey, records)
	}
	
	return records, nil
}

// LookupA overrides to use test server
func (tc *TestClient) LookupA(domain string) ([]net.IP, error) {
	cacheKey := fmt.Sprintf("A:%s", domain)
	
	// Check cache first
	if tc.config.EnableCaching {
		if cached := tc.getFromCache(cacheKey); cached != nil {
			tc.mu.Lock()
			tc.stats.Hits++
			tc.mu.Unlock()
			return cached.([]net.IP), nil
		}
	}
	
	// Use test server instead of real DNS
	ips, err := tc.testServer.LookupA(domain)
	
	tc.mu.Lock()
	if err != nil {
		tc.stats.Errors++
		tc.mu.Unlock()
		return nil, err
	}
	tc.stats.Misses++
	tc.mu.Unlock()
	
	// Cache the result
	if tc.config.EnableCaching {
		tc.setInCache(cacheKey, ips)
	}
	
	return ips, nil
}

// LookupMX overrides to use test server
func (tc *TestClient) LookupMX(domain string) ([]*net.MX, error) {
	cacheKey := fmt.Sprintf("MX:%s", domain)
	
	// Check cache first
	if tc.config.EnableCaching {
		if cached := tc.getFromCache(cacheKey); cached != nil {
			tc.mu.Lock()
			tc.stats.Hits++
			tc.mu.Unlock()
			return cached.([]*net.MX), nil
		}
	}
	
	// Use test server instead of real DNS
	mxRecords, err := tc.testServer.LookupMX(domain)
	
	tc.mu.Lock()
	if err != nil {
		tc.stats.Errors++
		tc.mu.Unlock()
		return nil, err
	}
	tc.stats.Misses++
	tc.mu.Unlock()
	
	// Cache the result
	if tc.config.EnableCaching {
		tc.setInCache(cacheKey, mxRecords)
	}
	
	return mxRecords, nil
}

// ParseTestConfig parses test configuration string
// Format: "domain:type:value:ttl_seconds"
func ParseTestConfig(config string) (domain, recordType, value string, ttl time.Duration, err error) {
	parts := strings.Split(config, ":")
	if len(parts) < 4 {
		return "", "", "", 0, fmt.Errorf("invalid format: expected domain:type:value:ttl_seconds")
	}
	
	domain = parts[0]
	recordType = strings.ToUpper(parts[1])
	value = parts[2]
	
	ttlSeconds, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", "", "", 0, fmt.Errorf("invalid TTL: %v", err)
	}
	
	ttl = time.Duration(ttlSeconds) * time.Second
	return domain, recordType, value, ttl, nil
} 