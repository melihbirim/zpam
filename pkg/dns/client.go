package dns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// DNSRecord represents a cached DNS record
type DNSRecord struct {
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// Client provides DNS lookup functionality with built-in caching
type Client struct {
	resolver *net.Resolver
	cache    map[string]*DNSRecord
	mu       sync.RWMutex
	config   Config
	stats    Stats
}

// Config contains DNS client configuration
type Config struct {
	Timeout       time.Duration `json:"timeout"`
	CacheSize     int           `json:"cache_size"`
	CacheTTL      time.Duration `json:"cache_ttl"`
	EnableCaching bool          `json:"enable_caching"`
}

// Stats tracks DNS client performance metrics
type Stats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Errors      int64     `json:"errors"`
	Entries     int64     `json:"entries"`
	Evictions   int64     `json:"evictions"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// NewClient creates a new DNS client with caching
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.CacheSize == 0 {
		config.CacheSize = 1000
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 30 * time.Minute
	}
	
	client := &Client{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: config.Timeout,
				}
				return d.DialContext(ctx, network, address)
			},
		},
		cache:  make(map[string]*DNSRecord),
		config: config,
		stats:  Stats{LastCleanup: time.Now()},
	}
	
	// Start background cleanup if caching is enabled
	if config.EnableCaching {
		go client.cleanupRoutine()
	}
	
	return client
}

// LookupTXT performs TXT record lookup with caching
func (c *Client) LookupTXT(domain string) ([]string, error) {
	cacheKey := fmt.Sprintf("TXT:%s", domain)
	
	// Check cache first
	if c.config.EnableCaching {
		if cached := c.getFromCache(cacheKey); cached != nil {
			c.mu.Lock()
			c.stats.Hits++
			c.mu.Unlock()
			return cached.([]string), nil
		}
	}
	
	// Perform DNS lookup
	ctx := context.Background()
	records, err := c.resolver.LookupTXT(ctx, domain)
	
	c.mu.Lock()
	if err != nil {
		c.stats.Errors++
		c.mu.Unlock()
		return nil, err
	}
	c.stats.Misses++
	c.mu.Unlock()
	
	// Cache the result
	if c.config.EnableCaching {
		c.setInCache(cacheKey, records)
	}
	
	return records, nil
}

// LookupA performs A record lookup with caching
func (c *Client) LookupA(domain string) ([]net.IP, error) {
	cacheKey := fmt.Sprintf("A:%s", domain)
	
	// Check cache first
	if c.config.EnableCaching {
		if cached := c.getFromCache(cacheKey); cached != nil {
			c.mu.Lock()
			c.stats.Hits++
			c.mu.Unlock()
			return cached.([]net.IP), nil
		}
	}
	
	// Perform DNS lookup
	ctx := context.Background()
	ipAddrs, err := c.resolver.LookupIPAddr(ctx, domain)
	
	c.mu.Lock()
	if err != nil {
		c.stats.Errors++
		c.mu.Unlock()
		return nil, err
	}
	c.stats.Misses++
	c.mu.Unlock()
	
	// Extract IPv4 addresses
	var ips []net.IP
	for _, addr := range ipAddrs {
		if ipv4 := addr.IP.To4(); ipv4 != nil {
			ips = append(ips, ipv4)
		}
	}
	
	// Cache the result
	if c.config.EnableCaching {
		c.setInCache(cacheKey, ips)
	}
	
	return ips, nil
}

// LookupMX performs MX record lookup with caching
func (c *Client) LookupMX(domain string) ([]*net.MX, error) {
	cacheKey := fmt.Sprintf("MX:%s", domain)
	
	// Check cache first
	if c.config.EnableCaching {
		if cached := c.getFromCache(cacheKey); cached != nil {
			c.mu.Lock()
			c.stats.Hits++
			c.mu.Unlock()
			return cached.([]*net.MX), nil
		}
	}
	
	// Perform DNS lookup
	ctx := context.Background()
	mxRecords, err := c.resolver.LookupMX(ctx, domain)
	
	c.mu.Lock()
	if err != nil {
		c.stats.Errors++
		c.mu.Unlock()
		return nil, err
	}
	c.stats.Misses++
	c.mu.Unlock()
	
	// Cache the result
	if c.config.EnableCaching {
		c.setInCache(cacheKey, mxRecords)
	}
	
	return mxRecords, nil
}

// LookupPTR performs PTR (reverse DNS) lookup with caching
func (c *Client) LookupPTR(ip string) ([]string, error) {
	cacheKey := fmt.Sprintf("PTR:%s", ip)
	
	// Check cache first
	if c.config.EnableCaching {
		if cached := c.getFromCache(cacheKey); cached != nil {
			c.mu.Lock()
			c.stats.Hits++
			c.mu.Unlock()
			return cached.([]string), nil
		}
	}
	
	// Perform DNS lookup
	ctx := context.Background()
	names, err := c.resolver.LookupAddr(ctx, ip)
	
	c.mu.Lock()
	if err != nil {
		c.stats.Errors++
		c.mu.Unlock()
		return nil, err
	}
	c.stats.Misses++
	c.mu.Unlock()
	
	// Cache the result
	if c.config.EnableCaching {
		c.setInCache(cacheKey, names)
	}
	
	return names, nil
}

// GetSPFRecord retrieves SPF record for domain
func (c *Client) GetSPFRecord(domain string) (string, error) {
	txtRecords, err := c.LookupTXT(domain)
	if err != nil {
		return "", err
	}
	
	// Find SPF record
	for _, record := range txtRecords {
		if len(record) > 7 && record[:7] == "v=spf1 " {
			return record, nil
		}
	}
	
	return "", fmt.Errorf("no SPF record found for domain %s", domain)
}

// GetDMARCRecord retrieves DMARC record for domain
func (c *Client) GetDMARCRecord(domain string) (string, error) {
	dmarcDomain := "_dmarc." + domain
	txtRecords, err := c.LookupTXT(dmarcDomain)
	if err != nil {
		return "", err
	}
	
	// Find DMARC record
	for _, record := range txtRecords {
		if len(record) > 8 && record[:8] == "v=DMARC1" {
			return record, nil
		}
	}
	
	return "", fmt.Errorf("no DMARC record found for domain %s", domain)
}

// CheckIPInA checks if IP matches any A record for domain
func (c *Client) CheckIPInA(domain, ip string) (bool, error) {
	ips, err := c.LookupA(domain)
	if err != nil {
		return false, err
	}
	
	targetIP := net.ParseIP(ip)
	if targetIP == nil {
		return false, fmt.Errorf("invalid IP address: %s", ip)
	}
	
	for _, domainIP := range ips {
		if domainIP.Equal(targetIP) {
			return true, nil
		}
	}
	
	return false, nil
}

// CheckIPInMX checks if IP matches any MX record for domain
func (c *Client) CheckIPInMX(domain, ip string) (bool, error) {
	mxRecords, err := c.LookupMX(domain)
	if err != nil {
		return false, err
	}
	
	for _, mx := range mxRecords {
		if matches, err := c.CheckIPInA(mx.Host, ip); err == nil && matches {
			return true, nil
		}
	}
	
	return false, nil
}

// ValidateReverseDNS checks if IP has valid reverse DNS
func (c *Client) ValidateReverseDNS(ip string) (bool, error) {
	names, err := c.LookupPTR(ip)
	return err == nil && len(names) > 0, err
}

// getFromCache retrieves value from cache if valid
func (c *Client) getFromCache(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	record, exists := c.cache[key]
	if !exists {
		return nil
	}
	
	// Check if expired
	if time.Now().After(record.ExpiresAt) {
		return nil
	}
	
	return record.Value
}

// setInCache stores value in cache
func (c *Client) setInCache(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check if we need to evict entries
	if len(c.cache) >= c.config.CacheSize {
		c.evictOldest()
	}
	
	now := time.Now()
	c.cache[key] = &DNSRecord{
		Type:      key[:3], // TXT, A, MX, PTR
		Value:     value,
		ExpiresAt: now.Add(c.config.CacheTTL),
		CreatedAt: now,
	}
	
	c.stats.Entries = int64(len(c.cache))
}

// evictOldest removes the oldest cache entry
func (c *Client) evictOldest() {
	if len(c.cache) == 0 {
		return
	}
	
	var oldestKey string
	var oldestTime time.Time
	
	for key, record := range c.cache {
		if oldestKey == "" || record.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = record.CreatedAt
		}
	}
	
	delete(c.cache, oldestKey)
	c.stats.Evictions++
}

// cleanupRoutine runs periodically to remove expired entries
func (c *Client) cleanupRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries from cache
func (c *Client) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for key, record := range c.cache {
		if now.After(record.ExpiresAt) {
			delete(c.cache, key)
			c.stats.Evictions++
		}
	}
	
	c.stats.Entries = int64(len(c.cache))
	c.stats.LastCleanup = now
}

// GetStats returns current performance statistics
func (c *Client) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := c.stats
	stats.Entries = int64(len(c.cache))
	return stats
}

// HitRate returns cache hit rate as percentage
func (c *Client) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}
	
	return float64(c.stats.Hits) / float64(total) * 100.0
}

// ClearCache removes all cached entries
func (c *Client) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.cache = make(map[string]*DNSRecord)
	c.stats.Entries = 0
}

// ResetStats resets performance statistics
func (c *Client) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.stats = Stats{
		Entries:     int64(len(c.cache)),
		LastCleanup: time.Now(),
	}
} 