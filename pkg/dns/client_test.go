package dns

import (
	"net"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// Test with default config
	client := NewClient(Config{})
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	
	if client.config.Timeout != 5*time.Second {
		t.Errorf("Expected default timeout 5s, got %v", client.config.Timeout)
	}
	
	if client.config.CacheSize != 1000 {
		t.Errorf("Expected default cache size 1000, got %d", client.config.CacheSize)
	}
	
	if client.config.CacheTTL != 30*time.Minute {
		t.Errorf("Expected default TTL 30m, got %v", client.config.CacheTTL)
	}
	
	// Test with custom config
	config := Config{
		Timeout:       2 * time.Second,
		CacheSize:     500,
		CacheTTL:      15 * time.Minute,
		EnableCaching: true,
	}
	
	client = NewClient(config)
	if client.config.Timeout != 2*time.Second {
		t.Errorf("Expected timeout 2s, got %v", client.config.Timeout)
	}
}

func TestClientStatistics(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       100 * time.Millisecond,
	})
	
	// Initial stats should be zero
	stats := client.GetStats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Entries != 0 {
		t.Error("Expected zero initial stats")
	}
	
	if client.HitRate() != 0.0 {
		t.Errorf("Expected 0%% hit rate, got %f%%", client.HitRate())
	}
}

func TestClientCacheOperations(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		CacheSize:     3,
		CacheTTL:      5 * time.Minute,
	})
	
	// Test cache storage and retrieval
	testData := []string{"record1", "record2"}
	client.setInCache("TEST:example.com", testData)
	
	// Should retrieve from cache
	cached := client.getFromCache("TEST:example.com")
	if cached == nil {
		t.Error("Expected cached data, got nil")
	}
	
	retrievedData := cached.([]string)
	if len(retrievedData) != 2 || retrievedData[0] != "record1" {
		t.Errorf("Expected cached data [record1, record2], got %v", retrievedData)
	}
	
	// Test cache expiry
	client.setInCache("EXPIRE:test.com", "data")
	
	// Manually expire the entry
	client.mu.Lock()
	for key, record := range client.cache {
		if key == "EXPIRE:test.com" {
			record.ExpiresAt = time.Now().Add(-1 * time.Minute)
		}
	}
	client.mu.Unlock()
	
	expired := client.getFromCache("EXPIRE:test.com")
	if expired != nil {
		t.Error("Expected expired data to return nil")
	}
}

func TestClientCacheEviction(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		CacheSize:     2, // Small cache for testing eviction
		CacheTTL:      5 * time.Minute,
	})
	
	// Fill cache to capacity
	client.setInCache("KEY1:domain1.com", "data1")
	client.setInCache("KEY2:domain2.com", "data2")
	
	stats := client.GetStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries, got %d", stats.Entries)
	}
	
	// Add another entry to trigger eviction
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	client.setInCache("KEY3:domain3.com", "data3")
	
	stats = client.GetStats()
	if stats.Entries != 2 {
		t.Errorf("Expected 2 entries after eviction, got %d", stats.Entries)
	}
	
	if stats.Evictions != 1 {
		t.Errorf("Expected 1 eviction, got %d", stats.Evictions)
	}
	
	// First entry should be evicted (oldest)
	cached := client.getFromCache("KEY1:domain1.com")
	if cached != nil {
		t.Error("Expected first entry to be evicted")
	}
	
	// Last entry should still exist
	cached = client.getFromCache("KEY3:domain3.com")
	if cached == nil {
		t.Error("Expected last entry to still exist")
	}
}

func TestClientClearAndReset(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       50 * time.Millisecond, // Short timeout for failing lookups
	})
	
	// Add some data and generate stats
	client.setInCache("TXT:cached.example.com", []string{"cached data"})
	
	// This should be a cache hit
	_, err := client.LookupTXT("cached.example.com")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// This should be a cache miss (and likely a DNS error due to short timeout)
	_, err = client.LookupTXT("nonexistent.invalid.domain.test")
	// Don't check for error since we expect DNS lookup to fail
	
	stats := client.GetStats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
	if stats.Entries != 1 {
		t.Errorf("Expected 1 entry, got %d", stats.Entries)
	}
	
	// Test clear cache
	client.ClearCache()
	stats = client.GetStats()
	if stats.Entries != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", stats.Entries)
	}
	if stats.Hits != 1 {
		t.Errorf("Clear should not affect hit stats, got %d", stats.Hits)
	}
	
	// Test reset stats
	client.ResetStats()
	stats = client.GetStats()
	if stats.Hits != 0 {
		t.Errorf("Reset should clear hit stats, got %d", stats.Hits)
	}
}

func TestClientHitRate(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       50 * time.Millisecond, // Short timeout for failing lookups
	})
	
	// Add data to cache
	client.setInCache("TXT:cached.example.com", []string{"cached data"})
	
	// Generate hits and misses using actual DNS lookup methods
	_, err := client.LookupTXT("cached.example.com") // hit
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	_, err = client.LookupTXT("cached.example.com") // hit
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// For a proper miss, we need to lookup an uncached domain (not an error)
	// Since real DNS lookups are unpredictable, let's just check the hit rate directly
	stats := client.GetStats()
	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	
	// The hit rate should be 100% since both lookups were cache hits
	hitRate := client.HitRate()
	if hitRate != 100.0 {
		t.Errorf("Expected 100%% hit rate, got %f%%", hitRate)
	}
}

func TestDNSHelperMethods(t *testing.T) {
	// Test IP validation
	testIP := "192.168.1.1"
	parsedIP := net.ParseIP(testIP)
	if parsedIP == nil {
		t.Errorf("Expected valid IP parsing for %s", testIP)
	}
	
	// Test invalid IP
	invalidIP := "not.an.ip"
	parsedIP = net.ParseIP(invalidIP)
	if parsedIP != nil {
		t.Errorf("Expected invalid IP parsing to return nil for %s", invalidIP)
	}
}

func TestGetSPFRecord(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       1 * time.Second,
	})
	
	// Mock TXT records
	client.setInCache("TXT:example.com", []string{
		"some other record",
		"v=spf1 include:_spf.example.com ~all",
		"another record",
	})
	
	spfRecord, err := client.GetSPFRecord("example.com")
	if err != nil {
		t.Errorf("Expected to find SPF record, got error: %v", err)
	}
	
	expected := "v=spf1 include:_spf.example.com ~all"
	if spfRecord != expected {
		t.Errorf("Expected SPF record '%s', got '%s'", expected, spfRecord)
	}
	
	// Test domain without SPF record
	client.setInCache("TXT:no-spf.com", []string{
		"some other record",
		"not an spf record",
	})
	
	_, err = client.GetSPFRecord("no-spf.com")
	if err == nil {
		t.Error("Expected error for domain without SPF record")
	}
}

func TestGetDMARCRecord(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       1 * time.Second,
	})
	
	// Mock DMARC TXT records
	client.setInCache("TXT:_dmarc.example.com", []string{
		"some other record",
		"v=DMARC1; p=quarantine; rua=mailto:reports@example.com",
	})
	
	dmarcRecord, err := client.GetDMARCRecord("example.com")
	if err != nil {
		t.Errorf("Expected to find DMARC record, got error: %v", err)
	}
	
	expected := "v=DMARC1; p=quarantine; rua=mailto:reports@example.com"
	if dmarcRecord != expected {
		t.Errorf("Expected DMARC record '%s', got '%s'", expected, dmarcRecord)
	}
	
	// Test domain without DMARC record
	client.setInCache("TXT:_dmarc.no-dmarc.com", []string{
		"some other record",
	})
	
	_, err = client.GetDMARCRecord("no-dmarc.com")
	if err == nil {
		t.Error("Expected error for domain without DMARC record")
	}
}

func TestCheckIPInA(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		Timeout:       1 * time.Second,
	})
	
	// Mock A records
	ips := []net.IP{
		net.ParseIP("192.168.1.1"),
		net.ParseIP("192.168.1.2"),
	}
	client.setInCache("A:example.com", ips)
	
	// Test IP that should match
	match, err := client.CheckIPInA("example.com", "192.168.1.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !match {
		t.Error("Expected IP to match A record")
	}
	
	// Test IP that should not match
	match, err = client.CheckIPInA("example.com", "10.0.0.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if match {
		t.Error("Expected IP to not match A record")
	}
	
	// Test invalid IP
	_, err = client.CheckIPInA("example.com", "invalid.ip")
	if err == nil {
		t.Error("Expected error for invalid IP")
	}
}

func TestPerformanceWithCaching(t *testing.T) {
	client := NewClient(Config{
		EnableCaching: true,
		CacheSize:     100,
		CacheTTL:      5 * time.Minute,
		Timeout:       100 * time.Millisecond,
	})
	
	// Pre-populate cache
	testData := []string{"test record"}
	client.setInCache("TXT:cached.example.com", testData)
	
	// First lookup (should be from cache)
	start := time.Now()
	records, err := client.LookupTXT("cached.example.com")
	elapsed1 := time.Since(start)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(records) != 1 || records[0] != "test record" {
		t.Errorf("Expected cached record, got %v", records)
	}
	
	// Second lookup (should also be from cache)
	start = time.Now()
	records, err = client.LookupTXT("cached.example.com")
	elapsed2 := time.Since(start)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Both should be very fast (under 1ms) since they're cached
	if elapsed1 > time.Millisecond || elapsed2 > time.Millisecond {
		t.Errorf("Cached lookups should be under 1ms, got %v and %v", elapsed1, elapsed2)
	}
	
	// Verify cache stats
	stats := client.GetStats()
	if stats.Hits < 2 {
		t.Errorf("Expected at least 2 cache hits, got %d", stats.Hits)
	}
} 