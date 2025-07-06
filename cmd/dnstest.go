package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/dns"
)

var (
	dnsTestPort     int
	dnsTestConfig   string
	dnsTestVerbose  bool
	dnsTestOutput   string
	dnsTestCount    int
	dnsTestDomains  []string
)

var dnsTestCmd = &cobra.Command{
	Use:   "dnstest",
	Short: "Internal DNS testing tools",
	Long:  `Manage internal DNS testing infrastructure for non-blocking DNS operations`,
}

// Server management commands
var dnsTestServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage DNS test server",
	Long:  `Start and manage the internal DNS test server for controlled testing`,
}

var dnsTestServerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start DNS test server",
	Long:  `Start the internal DNS test server with pre-loaded test data`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🧪 ZPO DNS Test Server\n")
		fmt.Printf("═══════════════════════════════════════\n")
		
		// Create test server
		testServer := dns.NewTestServer()
		
		// Load realistic test data
		fmt.Printf("📚 Loading test DNS data...\n")
		testServer.LoadTestData()
		
		fmt.Printf("✅ Test server initialized\n")
		fmt.Printf("📊 Records loaded: %d\n", testServer.GetRecordCount())
		
		if dnsTestVerbose {
			// Show loaded domains
			fmt.Printf("\n📋 Available test domains:\n")
			domains := []string{
				"gmail.com", "outlook.com", "yahoo.com",
				"government.gov", "scam-alert.biz", "test.org", 
				"phishing-site.net", "example.com",
			}
			for _, domain := range domains {
				fmt.Printf("  • %s\n", domain)
			}
		}
		
		fmt.Printf("\n🚀 DNS test server ready for testing\n")
		fmt.Printf("💡 Use 'zpo dnstest demo' to see performance comparison\n")
		fmt.Printf("📈 Use 'zpo dnstest benchmark' to run detailed benchmarks\n")
		
		return nil
	},
}

var dnsTestServerStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show DNS test server statistics",
	Long:  `Display current statistics from the DNS test server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This would connect to a running server
		fmt.Printf("📊 DNS Test Server Statistics\n")
		fmt.Printf("═══════════════════════════════════════\n")
		fmt.Printf("⚠️  Server statistics require a running test server\n")
		fmt.Printf("💡 Use 'zpo dnstest server start' to start the server\n")
		
		return nil
	},
}

// Demo and benchmark commands
var dnsTestDemoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Demonstrate DNS caching performance",
	Long:  `Run a demonstration comparing sync vs async DNS performance with caching`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🧪 ZPO DNS Performance Demonstration\n")
		fmt.Printf("═══════════════════════════════════════\n\n")
		
		// Create test server with data
		testServer := dns.NewTestServer()
		testServer.LoadTestData()
		
		// Test domains
		testDomains := []string{"gmail.com", "outlook.com", "yahoo.com", "government.gov"}
		
		fmt.Printf("1. Synchronous DNS Client Performance:\n")
		fmt.Printf("   Domain\t\t\tCold Cache\tWarm Cache\tSpeedup\n")
		fmt.Printf("   ------\t\t\t----------\t----------\t-------\n")
		
		// Create sync test client
		syncConfig := dns.Config{
			EnableCaching: true,
			CacheSize:     1000,
			CacheTTL:      30 * time.Minute,
			Timeout:       100 * time.Millisecond,
		}
		syncClient := dns.NewTestClient(syncConfig, testServer)
		
		for _, domain := range testDomains {
			// Cold cache
			syncClient.ClearCache()
			start := time.Now()
			_, err := syncClient.GetSPFRecord(domain)
			coldTime := time.Since(start)
			
			// Warm cache
			start = time.Now()
			_, err = syncClient.GetSPFRecord(domain)
			warmTime := time.Since(start)
			
			speedup := float64(coldTime.Nanoseconds()) / float64(warmTime.Nanoseconds())
			status := "✅"
			if err != nil {
				status = "❌"
			}
			
			fmt.Printf("   %-20s\t%v\t%v\t%.1fx %s\n", 
				domain, coldTime, warmTime, speedup, status)
		}
		
		fmt.Printf("\n2. Asynchronous DNS Client Performance:\n")
		fmt.Printf("   Testing concurrent lookups...\n")
		
		// Create async test client  
		asyncConfig := dns.Config{
			EnableCaching: true,
			CacheSize:     1000,
			CacheTTL:      30 * time.Minute,
			Timeout:       100 * time.Millisecond,
		}
		asyncClient := dns.NewAsyncClient(asyncConfig, 10)
		defer asyncClient.Stop()
		
		// Test async performance
		start := time.Now()
		var results []*dns.AsyncResult
		
		for _, domain := range testDomains {
			result := asyncClient.GetSPFRecordAsync(domain)
			results = append(results, result)
		}
		
		// Wait for all results
		for i, result := range results {
			_, err := result.Wait()
			status := "✅"
			cached := ""
			if result.IsFromCache() {
				cached = " (cached)"
			}
			if err != nil {
				status = "❌"
			}
			fmt.Printf("   %s: %s%s\n", testDomains[i], status, cached)
		}
		
		totalTime := time.Since(start)
		fmt.Printf("   Total time for %d concurrent lookups: %v\n", len(testDomains), totalTime)
		
		fmt.Printf("\n3. Cache Performance Analysis:\n")
		syncStats := syncClient.GetStats()
		asyncStats := asyncClient.GetStats()
		
		fmt.Printf("   Sync Client:\n")
		fmt.Printf("     Hit Rate: %.1f%% (%d hits / %d total)\n", 
			syncClient.HitRate(), syncStats.Hits, syncStats.Hits+syncStats.Misses+syncStats.Errors)
		fmt.Printf("     Cache Entries: %d\n", syncStats.Entries)
		
		fmt.Printf("   Async Client:\n")
		fmt.Printf("     Hit Rate: %.1f%% (%d hits / %d total)\n", 
			asyncClient.HitRate(), asyncStats.Hits, asyncStats.Hits+asyncStats.Misses+asyncStats.Errors)
		fmt.Printf("     Cache Entries: %d\n", asyncStats.Entries)
		
		fmt.Printf("\n4. Test Server Statistics:\n")
		serverStats := testServer.GetStats()
		fmt.Printf("   Total Queries: %d\n", serverStats.TotalQueries)
		fmt.Printf("   Cache Hits: %d\n", serverStats.CacheHits)
		fmt.Printf("   Record Misses: %d\n", serverStats.RecordMisses)
		fmt.Printf("   Expired Records: %d\n", serverStats.ExpiredRecords)
		
		fmt.Printf("\n✅ Demonstration Complete!\n")
		fmt.Printf("🚀 Key Benefits:\n")
		fmt.Printf("   • Instant cache hits (sub-microsecond performance)\n")
		fmt.Printf("   • Non-blocking async operations\n")
		fmt.Printf("   • Controlled test environment\n")
		fmt.Printf("   • Configurable TTL for realistic testing\n")
		
		return nil
	},
}

var dnsTestBenchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run DNS client benchmarks",
	Long:  `Run comprehensive benchmarks comparing different DNS client configurations`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("🧪 ZPO DNS Client Benchmark\n")
		fmt.Printf("═══════════════════════════════════════\n\n")
		
		// Create test server
		testServer := dns.NewTestServer()
		testServer.LoadTestData()
		
		// Test configurations
		configs := []struct {
			name        string
			config      dns.Config
			description string
		}{
			{
				name: "No Cache",
				config: dns.Config{
					EnableCaching: false,
					Timeout:       100 * time.Millisecond,
				},
				description: "Direct lookups without caching",
			},
			{
				name: "Small Cache",
				config: dns.Config{
					EnableCaching: true,
					CacheSize:     100,
					CacheTTL:      5 * time.Minute,
					Timeout:       100 * time.Millisecond,
				},
				description: "Small cache with 5min TTL",
			},
			{
				name: "Large Cache",
				config: dns.Config{
					EnableCaching: true,
					CacheSize:     5000,
					CacheTTL:      30 * time.Minute,
					Timeout:       100 * time.Millisecond,
				},
				description: "Large cache with 30min TTL",
			},
		}
		
		testDomains := []string{
			"gmail.com", "outlook.com", "yahoo.com", "government.gov",
			"scam-alert.biz", "test.org", "phishing-site.net", "example.com",
		}
		
		fmt.Printf("Testing %d configurations with %d domains\n\n", len(configs), len(testDomains))
		
		for _, cfg := range configs {
			fmt.Printf("📊 Configuration: %s\n", cfg.name)
			fmt.Printf("    %s\n", cfg.description)
			
			// Test sync client
			syncClient := dns.NewTestClient(cfg.config, testServer)
			
			// Measure cold performance
			var coldTimes []time.Duration
			for _, domain := range testDomains {
				syncClient.ClearCache()
				start := time.Now()
				_, err := syncClient.GetSPFRecord(domain)
				elapsed := time.Since(start)
				if err == nil {
					coldTimes = append(coldTimes, elapsed)
				}
			}
			
			// Measure warm performance
			var warmTimes []time.Duration
			for _, domain := range testDomains {
				start := time.Now()
				_, err := syncClient.GetSPFRecord(domain)
				elapsed := time.Since(start)
				if err == nil {
					warmTimes = append(warmTimes, elapsed)
				}
			}
			
			// Calculate statistics
			if len(coldTimes) > 0 && len(warmTimes) > 0 {
				avgCold := averageDuration(coldTimes)
				avgWarm := averageDuration(warmTimes)
				improvement := float64(avgCold.Nanoseconds()) / float64(avgWarm.Nanoseconds())
				
				fmt.Printf("    Cold Cache Avg: %v\n", avgCold)
				fmt.Printf("    Warm Cache Avg: %v\n", avgWarm)
				fmt.Printf("    Performance Improvement: %.1fx\n", improvement)
				
				stats := syncClient.GetStats()
				fmt.Printf("    Hit Rate: %.1f%%\n", syncClient.HitRate())
				fmt.Printf("    Cache Entries: %d\n", stats.Entries)
			}
			
			fmt.Printf("\n")
		}
		
		// Test async performance
		fmt.Printf("🚀 Async Client Performance:\n")
		asyncConfig := dns.Config{
			EnableCaching: true,
			CacheSize:     5000,
			CacheTTL:      30 * time.Minute,
			Timeout:       100 * time.Millisecond,
		}
		
		for workers := 1; workers <= 20; workers *= 2 {
			asyncClient := dns.NewAsyncClient(asyncConfig, workers)
			
			start := time.Now()
			var results []*dns.AsyncResult
			
			// Queue all lookups
			for _, domain := range testDomains {
				result := asyncClient.GetSPFRecordAsync(domain)
				results = append(results, result)
			}
			
			// Wait for completion
			for _, result := range results {
				result.Wait()
			}
			
			totalTime := time.Since(start)
			throughput := float64(len(testDomains)) / totalTime.Seconds()
			
			fmt.Printf("    %d workers: %v total (%.0f lookups/sec)\n", 
				workers, totalTime, throughput)
			
			asyncClient.Stop()
		}
		
		fmt.Printf("\n✅ Benchmark Complete!\n")
		
		return nil
	},
}

// Test data generation commands
var dnsTestGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate test emails with known DNS records",
	Long:  `Generate test emails using domains with known DNS records for reliable testing`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dnsTestOutput == "" {
			return fmt.Errorf("output directory is required")
		}
		
		fmt.Printf("🧪 Generating DNS Test Emails\n")
		fmt.Printf("═══════════════════════════════════════\n")
		fmt.Printf("📂 Output directory: %s\n", dnsTestOutput)
		fmt.Printf("📧 Email count: %d\n", dnsTestCount)
		fmt.Printf("\n")
		
		// Create output directory
		if err := os.MkdirAll(dnsTestOutput, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
		
		// Known domains with DNS records
		knownDomains := []string{
			"gmail.com", "outlook.com", "yahoo.com",
			"government.gov", "scam-alert.biz", "test.org",
		}
		
		spamDomains := []string{"scam-alert.biz", "phishing-site.net"}
		hamDomains := []string{"gmail.com", "outlook.com", "yahoo.com", "government.gov", "test.org"}
		
		generated := 0
		
		// Generate spam emails
		spamCount := dnsTestCount / 2
		for i := 0; i < spamCount; i++ {
			domain := spamDomains[i%len(spamDomains)]
			email := generateTestEmail(domain, true, i+1)
			filename := filepath.Join(dnsTestOutput, fmt.Sprintf("spam_%04d.eml", i+1))
			
			if err := os.WriteFile(filename, []byte(email), 0644); err != nil {
				return fmt.Errorf("failed to write spam email: %v", err)
			}
			generated++
		}
		
		// Generate ham emails
		hamCount := dnsTestCount - spamCount
		for i := 0; i < hamCount; i++ {
			domain := hamDomains[i%len(hamDomains)]
			email := generateTestEmail(domain, false, i+1)
			filename := filepath.Join(dnsTestOutput, fmt.Sprintf("ham_%04d.eml", i+1))
			
			if err := os.WriteFile(filename, []byte(email), 0644); err != nil {
				return fmt.Errorf("failed to write ham email: %v", err)
			}
			generated++
		}
		
		fmt.Printf("✅ Generated %d test emails\n", generated)
		fmt.Printf("🏷️  Domains used: %s\n", strings.Join(knownDomains, ", "))
		fmt.Printf("💡 These emails use domains with known DNS records for consistent testing\n")
		
		return nil
	},
}

// Configuration commands
var dnsTestConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Generate DNS test configuration",
	Long:  `Generate configuration files optimized for DNS testing`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "config-dnstest.yaml"
		if len(args) > 0 {
			configPath = args[0]
		}
		
		fmt.Printf("📝 Generating DNS test configuration: %s\n", configPath)
		
		config := generateDNSTestConfig()
		
		if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
			return fmt.Errorf("failed to write config: %v", err)
		}
		
		fmt.Printf("✅ DNS test configuration generated\n")
		fmt.Printf("🧪 Optimized for: Internal DNS testing with controlled TTL\n")
		fmt.Printf("⚡ Features: Non-blocking DNS, aggressive caching, fast timeouts\n")
		
		return nil
	},
}

// Helper functions

func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	var total int64
	for _, d := range durations {
		total += d.Nanoseconds()
	}
	
	return time.Duration(total / int64(len(durations)))
}

func generateTestEmail(domain string, isSpam bool, id int) string {
	timestamp := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	
	var from, subject, body string
	
	if isSpam {
		from = fmt.Sprintf("winner@%s", domain)
		subject = "URGENT!!! You have won $1,000,000!!!"
		body = `You have won our lottery! Claim your $1,000,000 prize now! Send your bank details to claim.

WARNING: IF YOU DON'T ACT NOW, YOU WILL REGRET IT FOR THE REST OF YOUR LIFE.

Click here immediately: http://phishing-site.net/click-here

Free money! Get rich quick! Act now! Limited time! Congratulations! You're our winner!

STOP STRUGGLING WITH FINANCIAL PROBLEMS. guaranteed income! make money fast!`
	} else {
		from = fmt.Sprintf("admin@%s", domain)
		subject = "Monthly system update notification"
		body = `Dear user,

This is a routine system maintenance notification. Our servers will undergo scheduled maintenance this weekend.

Expected downtime: 2 hours starting Saturday 2:00 AM UTC
Services affected: Email, web portal
Alternative access: Mobile app will remain available

Thank you for your patience.

Best regards,
IT Operations Team`
	}
	
	return fmt.Sprintf(`From: %s
To: user@example.com
Subject: %s
Date: %s
Message-ID: <%d.%d@dnstest.local>

%s`, from, subject, timestamp, id, time.Now().Unix(), body)
}

func generateDNSTestConfig() string {
	return `# ZPO Spam Filter Configuration - DNS Testing Mode
# Optimized for internal DNS testing with controlled environment

detection:
  spam_threshold: 4
  
  weights:
    subject_keywords: 3.0
    body_keywords: 2.0
    caps_ratio: 1.5
    exclamation_ratio: 1.0
    url_density: 2.5
    html_ratio: 1.0
    suspicious_headers: 2.0
    attachment_risk: 1.5
    domain_reputation: 3.0
    encoding_issues: 1.0
    from_to_mismatch: 2.0
    subject_length: 0.5
    frequency_penalty: 2.0
    word_frequency: 2.0
    header_validation: 2.5  # Full weight for DNS testing
  
  keywords:
    high_risk:
      - "free money"
      - "get rich quick"
      - "make money fast"
      - "guaranteed income"
      - "act now"
      - "limited time"
      - "urgent"
      - "congratulations"
      - "you have won"
      - "lottery"
      - "viagra"
      - "cialis"
    
    medium_risk:
      - "click here"
      - "special offer"
      - "discount"
      - "credit"
      - "loan"
      - "weight loss"
      - "earn extra"
    
    low_risk:
      - "free"
      - "offer"
      - "deal"
      - "sale"
      - "promotion"
      - "bonus"
  
  features:
    keyword_detection: true
    header_analysis: true
    attachment_scan: true
    domain_check: true
    frequency_tracking: true
    learning_mode: false

lists:
  whitelist_emails: []
  whitelist_domains: []
  blacklist_emails: []
  blacklist_domains: []
  
  trusted_domains:
    - "gmail.com"
    - "yahoo.com"
    - "outlook.com"
    - "government.gov"
    - "test.org"

performance:
  max_concurrent_emails: 20    # Higher concurrency for async testing
  timeout_ms: 500              # Fast timeout for test environment
  cache_size: 1000
  batch_size: 100

# DNS Testing Configuration - OPTIMIZED FOR INTERNAL TESTING
headers:
  enable_spf: true             # Enable SPF validation with test data
  enable_dkim: true            # Enable DKIM validation with test data
  enable_dmarc: true           # Enable DMARC validation with test data
  dns_timeout_ms: 100          # Very fast timeout for internal testing
  max_hop_count: 15
  suspicious_server_score: 75
  auth_weight: 2.5             # Full weight for authentication testing
  suspicious_weight: 2.0
  cache_size: 10000            # Large cache for extensive testing
  cache_ttl_min: 60            # Long TTL for test stability
  
  # DNS Testing Mode Settings
  use_internal_dns: true       # Enable internal DNS test server
  async_dns_workers: 10        # Worker pool size for async operations
  enable_async_dns: true       # Enable non-blocking DNS operations
  
  # Performance monitoring for testing
  enable_cache_stats: true
  log_cache_performance: true
  dns_performance_threshold_ms: 1  # Alert if DNS takes > 1ms (internal should be sub-ms)

# Word frequency learning
learning:
  enabled: true
  model_path: "zpo-model.json"
  min_word_length: 3
  max_word_length: 20
  case_sensitive: false
  spam_threshold: 0.7
  min_word_count: 2
  smoothing_factor: 1.0
  use_subject_words: true
  use_body_words: true
  use_header_words: false
  max_vocabulary_size: 10000
  auto_train: false

logging:
  level: "info"
  file: ""
  format: "text"
  max_size_mb: 10
  max_backups: 3`
}

func init() {
	// Add subcommands
	dnsTestCmd.AddCommand(dnsTestServerCmd)
	dnsTestCmd.AddCommand(dnsTestDemoCmd)
	dnsTestCmd.AddCommand(dnsTestBenchmarkCmd)
	dnsTestCmd.AddCommand(dnsTestGenerateCmd)
	dnsTestCmd.AddCommand(dnsTestConfigCmd)
	
	// Server subcommands
	dnsTestServerCmd.AddCommand(dnsTestServerStartCmd)
	dnsTestServerCmd.AddCommand(dnsTestServerStatsCmd)
	
	// Add flags
	dnsTestServerStartCmd.Flags().IntVarP(&dnsTestPort, "port", "p", 8053, "DNS server port")
	dnsTestServerStartCmd.Flags().BoolVarP(&dnsTestVerbose, "verbose", "v", false, "Verbose output")
	
	dnsTestGenerateCmd.Flags().StringVarP(&dnsTestOutput, "output", "o", "", "Output directory for generated emails")
	dnsTestGenerateCmd.Flags().IntVarP(&dnsTestCount, "count", "n", 20, "Number of emails to generate")
	dnsTestGenerateCmd.MarkFlagRequired("output")
	
	dnsTestBenchmarkCmd.Flags().BoolVarP(&dnsTestVerbose, "verbose", "v", false, "Verbose benchmark output")
	
	// Add to root command
	rootCmd.AddCommand(dnsTestCmd)
} 