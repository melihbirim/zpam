package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents ZPAM configuration
type Config struct {
	// Spam detection settings
	Detection DetectionConfig `yaml:"detection"`

	// Whitelist/Blacklist settings
	Lists ListsConfig `yaml:"lists"`

	// Performance settings
	Performance PerformanceConfig `yaml:"performance"`

	// Logging settings
	Logging LoggingConfig `yaml:"logging"`

	// Learning settings
	Learning LearningConfig `yaml:"learning"`

	// Headers validation settings
	Headers HeadersConfig `yaml:"headers"`

	// Milter server settings
	Milter MilterConfig `yaml:"milter"`

	// Plugin system settings
	Plugins PluginsConfig `yaml:"plugins"`
}

// DetectionConfig contains spam detection parameters
type DetectionConfig struct {
	// Scoring thresholds
	SpamThreshold int `yaml:"spam_threshold"` // 4-5 = spam

	// Feature weights
	Weights FeatureWeights `yaml:"weights"`

	// Keywords for detection
	Keywords KeywordLists `yaml:"keywords"`

	// Enable/disable features
	Features FeatureToggles `yaml:"features"`
}

// FeatureWeights defines scoring weights
type FeatureWeights struct {
	SubjectKeywords   float64 `yaml:"subject_keywords"`
	BodyKeywords      float64 `yaml:"body_keywords"`
	CapsRatio         float64 `yaml:"caps_ratio"`
	ExclamationRatio  float64 `yaml:"exclamation_ratio"`
	URLDensity        float64 `yaml:"url_density"`
	HTMLRatio         float64 `yaml:"html_ratio"`
	SuspiciousHeaders float64 `yaml:"suspicious_headers"`
	AttachmentRisk    float64 `yaml:"attachment_risk"`
	DomainReputation  float64 `yaml:"domain_reputation"`
	EncodingIssues    float64 `yaml:"encoding_issues"`
	FromToMismatch    float64 `yaml:"from_to_mismatch"`
	SubjectLength     float64 `yaml:"subject_length"`
	FrequencyPenalty  float64 `yaml:"frequency_penalty"`
	WordFrequency     float64 `yaml:"word_frequency"`
	HeaderValidation  float64 `yaml:"header_validation"`
}

// KeywordLists contains spam keyword categories
type KeywordLists struct {
	HighRisk   []string `yaml:"high_risk"`
	MediumRisk []string `yaml:"medium_risk"`
	LowRisk    []string `yaml:"low_risk"`
}

// FeatureToggles enables/disables detection features
type FeatureToggles struct {
	KeywordDetection  bool `yaml:"keyword_detection"`
	HeaderAnalysis    bool `yaml:"header_analysis"`
	AttachmentScan    bool `yaml:"attachment_scan"`
	DomainCheck       bool `yaml:"domain_check"`
	FrequencyTracking bool `yaml:"frequency_tracking"`
	LearningMode      bool `yaml:"learning_mode"`
}

// ListsConfig contains whitelist/blacklist settings
type ListsConfig struct {
	// Email addresses
	WhitelistEmails []string `yaml:"whitelist_emails"`
	BlacklistEmails []string `yaml:"blacklist_emails"`

	// Domains
	WhitelistDomains []string `yaml:"whitelist_domains"`
	BlacklistDomains []string `yaml:"blacklist_domains"`

	// Trusted domains (override other checks)
	TrustedDomains []string `yaml:"trusted_domains"`
}

// PerformanceConfig contains performance tuning
type PerformanceConfig struct {
	MaxConcurrentEmails int `yaml:"max_concurrent_emails"`
	TimeoutMs           int `yaml:"timeout_ms"`
	CacheSize           int `yaml:"cache_size"`
	BatchSize           int `yaml:"batch_size"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`  // debug, info, warn, error
	File       string `yaml:"file"`   // log file path, empty = stdout
	Format     string `yaml:"format"` // json, text
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
}

// LearningConfig contains word learning settings
type LearningConfig struct {
	// Enable word frequency learning
	Enabled bool `yaml:"enabled"`

	// Backend selection: "file" or "redis"
	Backend string `yaml:"backend"`

	// File-based backend settings
	File FileBackendConfig `yaml:"file"`

	// Redis-based backend settings
	Redis RedisBackendConfig `yaml:"redis"`

	// Training
	AutoTrain bool `yaml:"auto_train"`
}

// FileBackendConfig contains file-based learning settings
type FileBackendConfig struct {
	// Model file path
	ModelPath string `yaml:"model_path"`

	// Word processing
	MinWordLength int  `yaml:"min_word_length"`
	MaxWordLength int  `yaml:"max_word_length"`
	CaseSensitive bool `yaml:"case_sensitive"`

	// Learning parameters
	SpamThreshold   float64 `yaml:"spam_threshold"`
	MinWordCount    int     `yaml:"min_word_count"`
	SmoothingFactor float64 `yaml:"smoothing_factor"`

	// Features
	UseSubjectWords bool `yaml:"use_subject_words"`
	UseBodyWords    bool `yaml:"use_body_words"`
	UseHeaderWords  bool `yaml:"use_header_words"`

	// Performance
	MaxVocabularySize int `yaml:"max_vocabulary_size"`
}

// RedisBackendConfig contains Redis-based learning settings
type RedisBackendConfig struct {
	// Redis connection
	RedisURL    string `yaml:"redis_url"`
	KeyPrefix   string `yaml:"key_prefix"`
	DatabaseNum int    `yaml:"database_num"`

	// Tokenization (Rspamd-style OSB)
	OSBWindowSize  int `yaml:"osb_window_size"`
	MinTokenLength int `yaml:"min_token_length"`
	MaxTokenLength int `yaml:"max_token_length"`
	MaxTokens      int `yaml:"max_tokens"`

	// Learning parameters
	MinLearns     int     `yaml:"min_learns"`
	MaxLearns     int     `yaml:"max_learns"`
	SpamThreshold float64 `yaml:"spam_threshold"`

	// Per-user support
	PerUserStats bool   `yaml:"per_user_stats"`
	DefaultUser  string `yaml:"default_user"`

	// Token expiration (like Rspamd)
	TokenTTL        string `yaml:"token_ttl"`        // Duration string like "720h"
	CleanupInterval string `yaml:"cleanup_interval"` // Duration string like "6h"

	// Performance
	LocalCache bool   `yaml:"local_cache"`
	CacheTTL   string `yaml:"cache_ttl"` // Duration string like "5m"
	BatchSize  int    `yaml:"batch_size"`
}

// HeadersConfig contains email headers validation settings
type HeadersConfig struct {
	// Enable/disable validations
	EnableSPF   bool `yaml:"enable_spf"`
	EnableDKIM  bool `yaml:"enable_dkim"`
	EnableDMARC bool `yaml:"enable_dmarc"`

	// DNS timeout
	DNSTimeoutMs int `yaml:"dns_timeout_ms"`

	// Thresholds
	MaxHopCount           int `yaml:"max_hop_count"`
	SuspiciousServerScore int `yaml:"suspicious_server_score"`

	// Scoring weights
	AuthWeight       float64 `yaml:"auth_weight"`
	SuspiciousWeight float64 `yaml:"suspicious_weight"`

	// SpamAssassin-inspired penalties
	SPFFailPenalty      float64 `yaml:"spf_fail_penalty"`
	DKIMMissingPenalty  float64 `yaml:"dkim_missing_penalty"`
	DMARCMissingPenalty float64 `yaml:"dmarc_missing_penalty"`

	// Cache settings
	CacheSize   int `yaml:"cache_size"`
	CacheTTLMin int `yaml:"cache_ttl_min"`
}

// PluginsConfig contains plugin system settings
type PluginsConfig struct {
	// Global plugin settings
	Enabled       bool   `yaml:"enabled"`        // Enable plugin system
	Timeout       int    `yaml:"timeout_ms"`     // Plugin execution timeout
	MaxConcurrent int    `yaml:"max_concurrent"` // Max concurrent plugin executions
	ScoreMethod   string `yaml:"score_method"`   // How to combine scores: "weighted", "max", "average", "consensus"

	// Plugin-specific configurations
	SpamAssassin    PluginConfig `yaml:"spamassassin"`
	Rspamd          PluginConfig `yaml:"rspamd"`
	CustomRules     PluginConfig `yaml:"custom_rules"`
	VirusTotal      PluginConfig `yaml:"virustotal"`
	MachineLearning PluginConfig `yaml:"machine_learning"`
}

// PluginConfig contains individual plugin settings
type PluginConfig struct {
	Enabled  bool                   `yaml:"enabled"`
	Weight   float64                `yaml:"weight"`     // Weight for score combination
	Priority int                    `yaml:"priority"`   // Execution priority (lower = higher priority)
	Timeout  int                    `yaml:"timeout_ms"` // Individual plugin timeout
	Settings map[string]interface{} `yaml:"settings"`   // Plugin-specific settings
}

// MilterConfig contains milter server settings
type MilterConfig struct {
	// Enable milter server
	Enabled bool `yaml:"enabled"`

	// Network and address for milter socket
	Network string `yaml:"network"` // "tcp" or "unix"
	Address string `yaml:"address"` // "127.0.0.1:7357" or "/tmp/zpam.sock"

	// Connection settings
	ReadTimeoutMs  int `yaml:"read_timeout_ms"`
	WriteTimeoutMs int `yaml:"write_timeout_ms"`

	// Protocol options (what events to receive/skip)
	SkipConnect bool `yaml:"skip_connect"`
	SkipHelo    bool `yaml:"skip_helo"`
	SkipMail    bool `yaml:"skip_mail"`
	SkipRcpt    bool `yaml:"skip_rcpt"`
	SkipHeaders bool `yaml:"skip_headers"`
	SkipBody    bool `yaml:"skip_body"`
	SkipEOH     bool `yaml:"skip_eoh"`
	SkipData    bool `yaml:"skip_data"`

	// Actions (what modifications milter can perform)
	CanAddHeaders       bool `yaml:"can_add_headers"`
	CanChangeHeaders    bool `yaml:"can_change_headers"`
	CanAddRecipients    bool `yaml:"can_add_recipients"`
	CanRemoveRecipients bool `yaml:"can_remove_recipients"`
	CanChangeBody       bool `yaml:"can_change_body"`
	CanQuarantine       bool `yaml:"can_quarantine"`
	CanChangeFrom       bool `yaml:"can_change_from"`

	// Performance
	MaxConcurrentConnections int `yaml:"max_concurrent_connections"`
	GracefulShutdownTimeout  int `yaml:"graceful_shutdown_timeout_ms"`

	// Response modes
	RejectThreshold     int    `yaml:"reject_threshold"`     // Score >= this value gets rejected
	QuarantineThreshold int    `yaml:"quarantine_threshold"` // Score >= this value gets quarantined (if enabled)
	RejectMessage       string `yaml:"reject_message"`       // Custom rejection message
	QuarantineMessage   string `yaml:"quarantine_message"`   // Custom quarantine message

	// Header modifications
	AddSpamHeaders   bool   `yaml:"add_spam_headers"`   // Add X-Spam-* headers
	SpamHeaderPrefix string `yaml:"spam_header_prefix"` // Prefix for spam headers (default: "X-ZPAM-")
}

// DefaultConfig returns ZPAM default configuration
func DefaultConfig() *Config {
	return &Config{
		Detection: DetectionConfig{
			SpamThreshold: 4,
			Weights: FeatureWeights{
				SubjectKeywords:   3.0,
				BodyKeywords:      2.0,
				CapsRatio:         1.5,
				ExclamationRatio:  1.0,
				URLDensity:        2.5,
				HTMLRatio:         1.0,
				SuspiciousHeaders: 2.0,
				AttachmentRisk:    1.5,
				DomainReputation:  3.0,
				EncodingIssues:    1.0,
				FromToMismatch:    2.0,
				SubjectLength:     0.5,
				FrequencyPenalty:  2.0,
				WordFrequency:     2.0,
				HeaderValidation:  2.5,
			},
			Keywords: KeywordLists{
				HighRisk: []string{
					"free money", "get rich", "make money fast", "guaranteed income",
					"no risk", "act now", "limited time", "urgent", "congratulations",
					"you have won", "lottery", "inheritance", "nigerian prince",
					"viagra", "cialis", "pharmacy", "prescription",
				},
				MediumRisk: []string{
					"click here", "visit our website", "special offer", "discount",
					"save money", "credit", "loan", "mortgage", "insurance",
					"weight loss", "diet", "lose weight", "earn extra",
				},
				LowRisk: []string{
					"free", "offer", "deal", "sale", "promotion", "bonus",
					"gift", "prize", "winner", "selected", "opportunity",
				},
			},
			Features: FeatureToggles{
				KeywordDetection:  true,
				HeaderAnalysis:    true,
				AttachmentScan:    true,
				DomainCheck:       true,
				FrequencyTracking: true,
				LearningMode:      false,
			},
		},
		Lists: ListsConfig{
			WhitelistEmails:  []string{},
			BlacklistEmails:  []string{},
			WhitelistDomains: []string{},
			BlacklistDomains: []string{},
			TrustedDomains: []string{
				"gmail.com", "yahoo.com", "outlook.com", "hotmail.com",
				"apple.com", "microsoft.com", "google.com", "amazon.com",
			},
		},
		Performance: PerformanceConfig{
			MaxConcurrentEmails: 10,
			TimeoutMs:           5000, // 5 second timeout
			CacheSize:           1000,
			BatchSize:           100,
		},
		Logging: LoggingConfig{
			Level:      "info",
			File:       "",
			Format:     "text",
			MaxSizeMB:  10,
			MaxBackups: 3,
		},
		Learning: LearningConfig{
			Enabled:   false,
			Backend:   "file",
			AutoTrain: false,
			File: FileBackendConfig{
				ModelPath:         "zpam-model.json",
				MinWordLength:     3,
				MaxWordLength:     20,
				CaseSensitive:     false,
				SpamThreshold:     0.7,
				MinWordCount:      2,
				SmoothingFactor:   1.0,
				UseSubjectWords:   true,
				UseBodyWords:      true,
				UseHeaderWords:    false,
				MaxVocabularySize: 10000,
			},
			Redis: RedisBackendConfig{
				RedisURL:        "redis://localhost:6379",
				KeyPrefix:       "zpam:bayes",
				DatabaseNum:     0,
				OSBWindowSize:   5,
				MinTokenLength:  3,
				MaxTokenLength:  32,
				MaxTokens:       1000,
				MinLearns:       200,
				MaxLearns:       5000,
				SpamThreshold:   0.95,
				PerUserStats:    true,
				DefaultUser:     "global",
				TokenTTL:        "720h",
				CleanupInterval: "6h",
				LocalCache:      true,
				CacheTTL:        "5m",
				BatchSize:       100,
			},
		},
		Headers: HeadersConfig{
			EnableSPF:             true,
			EnableDKIM:            true,
			EnableDMARC:           true,
			DNSTimeoutMs:          5000,
			MaxHopCount:           15,
			SuspiciousServerScore: 75,
			AuthWeight:            2.0,
			SuspiciousWeight:      2.5,
			SPFFailPenalty:        0.9, // SpamAssassin-inspired
			DKIMMissingPenalty:    1.0, // Much more reasonable
			DMARCMissingPenalty:   1.5, // Moderate penalty
			CacheSize:             1000,
			CacheTTLMin:           60,
		},
		Milter: MilterConfig{
			Enabled:                  false,
			Network:                  "tcp",
			Address:                  "127.0.0.1:7357",
			ReadTimeoutMs:            10000,
			WriteTimeoutMs:           10000,
			SkipConnect:              false,
			SkipHelo:                 false,
			SkipMail:                 false,
			SkipRcpt:                 false,
			SkipHeaders:              false,
			SkipBody:                 false,
			SkipEOH:                  false,
			SkipData:                 false,
			CanAddHeaders:            true,
			CanChangeHeaders:         true,
			CanAddRecipients:         true,
			CanRemoveRecipients:      false,
			CanChangeBody:            true,
			CanQuarantine:            false,
			CanChangeFrom:            true,
			MaxConcurrentConnections: 10,
			GracefulShutdownTimeout:  10000,
			RejectThreshold:          5,
			QuarantineThreshold:      4,
			RejectMessage:            "",
			QuarantineMessage:        "",
			AddSpamHeaders:           true,
			SpamHeaderPrefix:         "X-ZPAM-",
		},
		Plugins: PluginsConfig{
			Enabled:       false,      // Disabled by default
			Timeout:       5000,       // 5 seconds timeout
			MaxConcurrent: 3,          // Max 3 plugins running simultaneously
			ScoreMethod:   "weighted", // Use weighted scoring by default
			SpamAssassin: PluginConfig{
				Enabled:  false,
				Weight:   2.0,
				Priority: 1,
				Timeout:  5000,
				Settings: map[string]interface{}{
					"executable": "spamc",
					"host":       "localhost",
					"port":       783,
					"max_size":   10485760, // 10MB
				},
			},
			Rspamd: PluginConfig{
				Enabled:  false,
				Weight:   2.0,
				Priority: 2,
				Timeout:  3000,
				Settings: map[string]interface{}{
					"url":      "http://localhost:11334",
					"password": "",
				},
			},
			CustomRules: PluginConfig{
				Enabled:  false,
				Weight:   1.5,
				Priority: 3,
				Timeout:  1000,
				Settings: map[string]interface{}{
					"rules": []interface{}{},
				},
			},
			VirusTotal: PluginConfig{
				Enabled:  false,
				Weight:   3.0,
				Priority: 4,
				Timeout:  10000,
				Settings: map[string]interface{}{
					"api_key": "",
					"timeout": 10000,
				},
			},
			MachineLearning: PluginConfig{
				Enabled:  false,
				Weight:   2.5,
				Priority: 5,
				Timeout:  5000,
				Settings: map[string]interface{}{
					"model_path": "",
					"threshold":  0.7,
				},
			},
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// If no config file specified, return defaults
	if configPath == "" {
		return config, nil
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse YAML
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return config, nil
}

// SaveConfig saves configuration to file
func (c *Config) SaveConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write to file
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate spam threshold
	if c.Detection.SpamThreshold < 1 || c.Detection.SpamThreshold > 5 {
		return fmt.Errorf("spam_threshold must be between 1 and 5")
	}

	// Validate performance settings
	if c.Performance.MaxConcurrentEmails < 1 {
		return fmt.Errorf("max_concurrent_emails must be >= 1")
	}

	if c.Performance.TimeoutMs < 100 {
		return fmt.Errorf("timeout_ms must be >= 100")
	}

	// Validate logging level
	validLevels := []string{"debug", "info", "warn", "error"}
	validLevel := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			validLevel = true
			break
		}
	}
	if !validLevel {
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	// Validate milter settings
	if c.Milter.Enabled {
		if c.Milter.Network != "tcp" && c.Milter.Network != "unix" {
			return fmt.Errorf("milter network must be 'tcp' or 'unix'")
		}

		if c.Milter.Address == "" {
			return fmt.Errorf("milter address cannot be empty when enabled")
		}

		if c.Milter.ReadTimeoutMs < 1000 {
			return fmt.Errorf("milter read_timeout_ms must be >= 1000")
		}

		if c.Milter.WriteTimeoutMs < 1000 {
			return fmt.Errorf("milter write_timeout_ms must be >= 1000")
		}

		if c.Milter.MaxConcurrentConnections < 1 {
			return fmt.Errorf("milter max_concurrent_connections must be >= 1")
		}

		if c.Milter.RejectThreshold < 1 || c.Milter.RejectThreshold > 5 {
			return fmt.Errorf("milter reject_threshold must be between 1 and 5")
		}

		if c.Milter.QuarantineThreshold < 1 || c.Milter.QuarantineThreshold > 5 {
			return fmt.Errorf("milter quarantine_threshold must be between 1 and 5")
		}

		if c.Milter.CanQuarantine && c.Milter.QuarantineThreshold >= c.Milter.RejectThreshold {
			return fmt.Errorf("milter quarantine_threshold must be less than reject_threshold")
		}
	}

	return nil
}

// IsWhitelisted checks if email/domain is whitelisted
func (c *Config) IsWhitelisted(email, domain string) bool {
	// Check email whitelist
	for _, whiteEmail := range c.Lists.WhitelistEmails {
		if email == whiteEmail {
			return true
		}
	}

	// Check domain whitelist
	for _, whiteDomain := range c.Lists.WhitelistDomains {
		if domain == whiteDomain {
			return true
		}
	}

	return false
}

// IsBlacklisted checks if email/domain is blacklisted
func (c *Config) IsBlacklisted(email, domain string) bool {
	// Check email blacklist
	for _, blackEmail := range c.Lists.BlacklistEmails {
		if email == blackEmail {
			return true
		}
	}

	// Check domain blacklist
	for _, blackDomain := range c.Lists.BlacklistDomains {
		if domain == blackDomain {
			return true
		}
	}

	return false
}

// IsTrustedDomain checks if domain is trusted
func (c *Config) IsTrustedDomain(domain string) bool {
	for _, trustedDomain := range c.Lists.TrustedDomains {
		if domain == trustedDomain {
			return true
		}
	}
	return false
}
