package plugins

import (
	"context"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

// PluginResult represents the result from a plugin execution
type PluginResult struct {
	Name        string            `json:"name"`
	Score       float64           `json:"score"`
	Confidence  float64           `json:"confidence"` // 0.0-1.0
	Details     map[string]any    `json:"details"`
	ProcessTime time.Duration     `json:"process_time"`
	Error       error             `json:"error,omitempty"`
	Rules       []string          `json:"rules,omitempty"`    // Triggered rules
	Metadata    map[string]string `json:"metadata,omitempty"` // Additional metadata
}

// PluginConfig represents plugin-specific configuration
type PluginConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Weight   float64        `yaml:"weight"`   // Score multiplier (0.0-5.0)
	Timeout  time.Duration  `yaml:"timeout"`  // Max execution time
	Priority int            `yaml:"priority"` // Execution order (lower = earlier)
	Settings map[string]any `yaml:"settings"` // Plugin-specific settings
}

// Plugin represents the base interface for all spam detection plugins
type Plugin interface {
	// Name returns the plugin name (e.g., "spamassassin", "rspamd", "custom_rules")
	Name() string

	// Version returns the plugin version
	Version() string

	// Description returns a brief description of what the plugin does
	Description() string

	// Initialize sets up the plugin with configuration
	Initialize(config *PluginConfig) error

	// IsHealthy checks if the plugin is ready to process emails
	IsHealthy(ctx context.Context) error

	// Cleanup releases any resources
	Cleanup() error
}

// ContentAnalyzer plugins analyze email content for spam indicators
type ContentAnalyzer interface {
	Plugin

	// AnalyzeContent examines email content and returns spam score
	AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error)
}

// ReputationChecker plugins check sender/domain/URL reputation
type ReputationChecker interface {
	Plugin

	// CheckReputation verifies reputation of email components
	CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error)
}

// AttachmentScanner plugins scan email attachments
type AttachmentScanner interface {
	Plugin

	// ScanAttachments analyzes email attachments for threats
	ScanAttachments(ctx context.Context, attachments []email.Attachment) (*PluginResult, error)
}

// MLClassifier plugins use machine learning for classification
type MLClassifier interface {
	Plugin

	// Classify uses ML models to classify email as spam/ham
	Classify(ctx context.Context, email *email.Email) (*PluginResult, error)

	// Train updates the ML model with new data (optional)
	Train(ctx context.Context, emails []email.Email, labels []bool) error
}

// ExternalEngine plugins integrate with external spam detection engines
type ExternalEngine interface {
	Plugin

	// Analyze sends email to external engine for analysis
	Analyze(ctx context.Context, email *email.Email) (*PluginResult, error)

	// GetEngineStats returns statistics from the external engine
	GetEngineStats(ctx context.Context) (map[string]any, error)
}

// CustomRuleEngine plugins allow user-defined rules
type CustomRuleEngine interface {
	Plugin

	// EvaluateRules runs custom rules against email
	EvaluateRules(ctx context.Context, email *email.Email) (*PluginResult, error)

	// LoadRules loads rules from configuration
	LoadRules(rules []Rule) error
}

// Rule represents a custom spam detection rule
type Rule struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Enabled     bool              `yaml:"enabled"`
	Score       float64           `yaml:"score"`
	Conditions  []RuleCondition   `yaml:"conditions"`
	Actions     []RuleAction      `yaml:"actions"`
	Metadata    map[string]string `yaml:"metadata"`
}

// RuleCondition defines what to check in an email
type RuleCondition struct {
	Type          string `yaml:"type"`     // "subject", "body", "header", "from", "attachment"
	Operator      string `yaml:"operator"` // "contains", "matches", "equals", "regex"
	Value         string `yaml:"value"`    // Pattern to match
	CaseSensitive bool   `yaml:"case_sensitive"`
}

// RuleAction defines what to do when a rule matches
type RuleAction struct {
	Type  string `yaml:"type"`  // "score", "tag", "log", "block"
	Value string `yaml:"value"` // Action-specific value
}

// PluginRegistry manages all available plugins
type PluginRegistry interface {
	// Register adds a plugin to the registry
	Register(plugin Plugin) error

	// Get retrieves a plugin by name
	Get(name string) (Plugin, error)

	// List returns all registered plugins
	List() []Plugin

	// GetByType returns plugins implementing specific interface
	GetByType(pluginType string) ([]Plugin, error)

	// IsEnabled checks if a plugin is enabled
	IsEnabled(name string) bool
}

// PluginManager orchestrates plugin execution
type PluginManager interface {
	// LoadPlugins loads and initializes all configured plugins
	LoadPlugins(configs map[string]*PluginConfig) error

	// ExecuteAll runs all enabled plugins on an email
	ExecuteAll(ctx context.Context, email *email.Email) ([]*PluginResult, error)

	// ExecuteByType runs plugins of specific type
	ExecuteByType(ctx context.Context, email *email.Email, pluginType string) ([]*PluginResult, error)

	// CombineScores aggregates plugin results into final score
	CombineScores(results []*PluginResult) (float64, error)

	// GetStats returns plugin execution statistics
	GetStats() map[string]PluginStats

	// Shutdown gracefully shuts down all plugins
	Shutdown(ctx context.Context) error
}

// PluginStats tracks plugin performance metrics
type PluginStats struct {
	Name           string        `json:"name"`
	ExecutionCount int64         `json:"execution_count"`
	TotalTime      time.Duration `json:"total_time"`
	AverageTime    time.Duration `json:"average_time"`
	ErrorCount     int64         `json:"error_count"`
	LastExecution  time.Time     `json:"last_execution"`
	SuccessRate    float64       `json:"success_rate"`
}

// ScoreAggregation defines how to combine multiple plugin scores
type ScoreAggregation struct {
	Method    string             `yaml:"method"`    // "weighted_sum", "max", "average", "consensus"
	Weights   map[string]float64 `yaml:"weights"`   // Plugin name -> weight
	Threshold float64            `yaml:"threshold"` // Minimum score for spam classification
}

// PluginEvent represents plugin lifecycle events
type PluginEvent struct {
	Type      string    `json:"type"`   // "loaded", "executed", "error", "unloaded"
	Plugin    string    `json:"plugin"` // Plugin name
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
	Error     error     `json:"error,omitempty"`
}

// EventHandler handles plugin events
type EventHandler interface {
	HandleEvent(event *PluginEvent)
}
