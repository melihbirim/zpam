package plugins

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/zpam/spam-filter/pkg/email"
	"gopkg.in/yaml.v3"
)

// CustomRulesConfig represents the structure of the external custom_rules.yml file
type CustomRulesConfig struct {
	Settings struct {
		Enabled          bool `yaml:"enabled"`
		CaseSensitive    bool `yaml:"case_sensitive"`
		LogMatches       bool `yaml:"log_matches"`
		MaxRulesPerEmail int  `yaml:"max_rules_per_email"`
	} `yaml:"settings"`
	Rules    []Rule `yaml:"rules"`
	RuleSets map[string]struct {
		Enabled bool     `yaml:"enabled"`
		Rules   []string `yaml:"rules"`
	} `yaml:"rule_sets"`
	Advanced struct {
		CombineScores      bool     `yaml:"combine_scores"`
		MaxTotalScore      float64  `yaml:"max_total_score"`
		RuleTimeoutMs      int      `yaml:"rule_timeout_ms"`
		ParallelExecution  bool     `yaml:"parallel_execution"`
		WhitelistedDomains []string `yaml:"whitelisted_domains"`
		Learning           struct {
			Enabled           bool `yaml:"enabled"`
			AutoAdjustScores  bool `yaml:"auto_adjust_scores"`
			FeedbackThreshold int  `yaml:"feedback_threshold"`
		} `yaml:"learning"`
	} `yaml:"advanced"`
}

// CustomRulesPlugin implements user-defined spam detection rules
type CustomRulesPlugin struct {
	config      *PluginConfig
	enabled     bool
	rules       []Rule
	rulesConfig *CustomRulesConfig
	stats       map[string]int // Rule ID -> trigger count
	rulesFile   string
}

// NewCustomRulesPlugin creates a new custom rules plugin
func NewCustomRulesPlugin() *CustomRulesPlugin {
	return &CustomRulesPlugin{
		enabled:   false,
		rules:     []Rule{},
		stats:     make(map[string]int),
		rulesFile: "custom_rules.yml", // Default rules file
	}
}

// Name returns the plugin name
func (cr *CustomRulesPlugin) Name() string {
	return "custom_rules"
}

// Version returns the plugin version
func (cr *CustomRulesPlugin) Version() string {
	return "1.0.0"
}

// Description returns plugin description
func (cr *CustomRulesPlugin) Description() string {
	return "Custom rules engine for user-defined spam detection logic"
}

// Initialize sets up the plugin with configuration
func (cr *CustomRulesPlugin) Initialize(config *PluginConfig) error {
	cr.config = config
	cr.enabled = config.Enabled

	if !cr.enabled {
		return nil
	}

	// Check if custom rules file path is specified in settings
	if settings := config.Settings; settings != nil {
		if rulesFile, ok := settings["rules_file"].(string); ok && rulesFile != "" {
			cr.rulesFile = rulesFile
		}
	}

	// Load rules from external file
	if err := cr.loadRulesFromFile(); err != nil {
		return fmt.Errorf("failed to load custom rules: %v", err)
	}

	return nil
}

// loadRulesFromFile loads rules from the external YAML file
func (cr *CustomRulesPlugin) loadRulesFromFile() error {
	// Check if file exists
	if _, err := os.Stat(cr.rulesFile); os.IsNotExist(err) {
		// If file doesn't exist, create it with default content
		if err := cr.createDefaultRulesFile(); err != nil {
			return fmt.Errorf("failed to create default rules file: %v", err)
		}
	}

	// Read the YAML file
	data, err := os.ReadFile(cr.rulesFile)
	if err != nil {
		return fmt.Errorf("failed to read rules file %s: %v", cr.rulesFile, err)
	}

	// Parse YAML
	var rulesConfig CustomRulesConfig
	if err := yaml.Unmarshal(data, &rulesConfig); err != nil {
		return fmt.Errorf("failed to parse rules file %s: %v", cr.rulesFile, err)
	}

	cr.rulesConfig = &rulesConfig

	// Load only enabled rules
	cr.rules = []Rule{}
	if rulesConfig.Settings.Enabled {
		for _, rule := range rulesConfig.Rules {
			if rule.Enabled {
				cr.rules = append(cr.rules, rule)
			}
		}
	}

	// Load rule sets if any are enabled
	for setName, ruleSet := range rulesConfig.RuleSets {
		if ruleSet.Enabled {
			cr.loadRuleSet(setName, ruleSet.Rules, rulesConfig.Rules)
		}
	}

	// Clear stats for new rules
	cr.stats = make(map[string]int)

	return nil
}

// loadRuleSet loads a specific rule set
func (cr *CustomRulesPlugin) loadRuleSet(setName string, ruleIDs []string, allRules []Rule) {
	ruleMap := make(map[string]Rule)
	for _, rule := range allRules {
		ruleMap[rule.ID] = rule
	}

	for _, ruleID := range ruleIDs {
		if rule, exists := ruleMap[ruleID]; exists && rule.Enabled {
			// Add rule set prefix to distinguish from regular rules
			rule.ID = fmt.Sprintf("%s_%s", setName, rule.ID)
			rule.Name = fmt.Sprintf("[%s] %s", setName, rule.Name)
			cr.rules = append(cr.rules, rule)
		}
	}
}

// createDefaultRulesFile creates a basic default rules file if none exists
func (cr *CustomRulesPlugin) createDefaultRulesFile() error {
	defaultConfig := `# ZPAM Custom Rules Configuration
# This file defines custom spam detection rules
# Edit this file to add your own rules

settings:
  enabled: true
  case_sensitive: false
  log_matches: true
  max_rules_per_email: 50

rules:
  - id: congratulations_spam
    name: Congratulations Spam
    description: Detect congratulations-based scams
    enabled: true
    score: 8.0
    conditions:
      - type: subject
        operator: contains
        value: congratulations
        case_sensitive: false
    actions:
      - type: tag
        value: congratulations_scam
      - type: log
        value: Congratulations scam detected

  - id: urgent_action
    name: Urgent Action
    description: Detect urgent action pressure tactics
    enabled: true
    score: 5.0
    conditions:
      - type: subject
        operator: regex
        value: (urgent|act now|limited time)
        case_sensitive: false
    actions:
      - type: tag
        value: urgent_pressure
      - type: log
        value: Urgent pressure tactic detected

rule_sets:
  financial_strict:
    enabled: false
    rules:
      - congratulations_spam
      - urgent_action

advanced:
  combine_scores: true
  max_total_score: 50.0
  rule_timeout_ms: 100
  parallel_execution: true
  whitelisted_domains: []
  learning:
    enabled: false
    auto_adjust_scores: false
    feedback_threshold: 10
`

	return os.WriteFile(cr.rulesFile, []byte(defaultConfig), 0644)
}

// IsHealthy checks if the plugin is ready
func (cr *CustomRulesPlugin) IsHealthy(ctx context.Context) error {
	if !cr.enabled {
		return fmt.Errorf("custom rules plugin not enabled")
	}

	if cr.rulesConfig == nil {
		return fmt.Errorf("rules configuration not loaded")
	}

	if len(cr.rules) == 0 {
		return fmt.Errorf("no rules loaded")
	}

	return nil
}

// Cleanup releases resources
func (cr *CustomRulesPlugin) Cleanup() error {
	// Clear stats
	cr.stats = make(map[string]int)
	return nil
}

// LoadRules implements CustomRuleEngine interface (for backward compatibility)
func (cr *CustomRulesPlugin) LoadRules(rules []Rule) error {
	cr.rules = rules
	cr.stats = make(map[string]int)
	return nil
}

// ReloadRules reloads rules from the external file
func (cr *CustomRulesPlugin) ReloadRules() error {
	return cr.loadRulesFromFile()
}

// EvaluateRules implements CustomRuleEngine interface
func (cr *CustomRulesPlugin) EvaluateRules(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	result := &PluginResult{
		Name:        cr.Name(),
		Score:       0,
		Confidence:  0.7, // Custom rules are generally reliable
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
		Rules:       []string{},
		ProcessTime: 0,
	}

	if !cr.enabled || cr.rulesConfig == nil || !cr.rulesConfig.Settings.Enabled {
		result.Error = fmt.Errorf("plugin not enabled or configured")
		result.ProcessTime = time.Since(start)
		return result, nil
	}

	// Check if sender domain is whitelisted
	if cr.isDomainWhitelisted(email.From) {
		result.Metadata["whitelisted"] = "true"
		result.ProcessTime = time.Since(start)
		return result, nil
	}

	var triggeredRules []string
	var totalScore float64
	var ruleCount int
	maxRules := cr.rulesConfig.Settings.MaxRulesPerEmail

	// Evaluate each rule
	for _, rule := range cr.rules {
		if !rule.Enabled {
			continue
		}

		// Stop if max rules per email reached
		if maxRules > 0 && ruleCount >= maxRules {
			break
		}

		matched, err := cr.evaluateRule(rule, email)
		if err != nil {
			// Log error but continue with other rules
			continue
		}

		if matched {
			triggeredRules = append(triggeredRules, fmt.Sprintf("%s (%.1f)", rule.Name, rule.Score))
			totalScore += rule.Score
			ruleCount++

			// Update statistics
			cr.stats[rule.ID]++

			// Execute rule actions
			cr.executeRuleActions(rule, result)

			// Log if enabled
			if cr.rulesConfig.Settings.LogMatches {
				result.Details["matched_rules"] = append(
					result.Details["matched_rules"].([]string),
					fmt.Sprintf("%s: %s", rule.ID, rule.Description),
				)
			}
		}
	}

	// Apply advanced settings
	if cr.rulesConfig.Advanced.CombineScores {
		if cr.rulesConfig.Advanced.MaxTotalScore > 0 && totalScore > cr.rulesConfig.Advanced.MaxTotalScore {
			totalScore = cr.rulesConfig.Advanced.MaxTotalScore
		}
	}

	result.Score = totalScore
	result.Rules = triggeredRules
	result.ProcessTime = time.Since(start)

	// Calculate confidence based on number of rules triggered
	if ruleCount > 0 {
		confidence := 0.5 + float64(ruleCount)*0.1
		if confidence > 1.0 {
			confidence = 1.0
		}
		result.Confidence = confidence
	}

	// Add metadata
	result.Metadata["rules_triggered"] = fmt.Sprintf("%d", ruleCount)
	result.Metadata["total_rules"] = fmt.Sprintf("%d", len(cr.rules))
	result.Metadata["rules_file"] = cr.rulesFile

	return result, nil
}

// isDomainWhitelisted checks if a domain is whitelisted
func (cr *CustomRulesPlugin) isDomainWhitelisted(email string) bool {
	if cr.rulesConfig == nil {
		return false
	}

	// Extract domain from email
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])

	// Remove angle brackets if present
	domain = strings.Trim(domain, "<>")

	for _, whitelistedDomain := range cr.rulesConfig.Advanced.WhitelistedDomains {
		if strings.ToLower(whitelistedDomain) == domain {
			return true
		}
	}

	return false
}

// evaluateRule checks if a rule matches the email
func (cr *CustomRulesPlugin) evaluateRule(rule Rule, email *email.Email) (bool, error) {
	// All conditions must match for the rule to trigger
	for _, condition := range rule.Conditions {
		matched, err := cr.evaluateCondition(condition, email)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil // Short circuit - all conditions must match
		}
	}

	return len(rule.Conditions) > 0, nil // Only match if there are conditions
}

// evaluateCondition checks if a single condition matches
func (cr *CustomRulesPlugin) evaluateCondition(condition RuleCondition, email *email.Email) (bool, error) {
	var text string

	// Get the text to evaluate based on condition type
	switch strings.ToLower(condition.Type) {
	case "subject":
		text = email.Subject
	case "body":
		text = email.Body
	case "from":
		text = email.From
	case "to":
		text = strings.Join(email.To, ", ")
	case "header":
		// For headers, expect condition.Value to be in format "HeaderName:pattern"
		parts := strings.SplitN(condition.Value, ":", 2)
		if len(parts) != 2 {
			return false, fmt.Errorf("header condition must be in format 'HeaderName:pattern'")
		}
		headerName := parts[0]
		condition.Value = parts[1] // Update value to just the pattern
		if headerValue, exists := email.Headers[headerName]; exists {
			text = headerValue
		} else {
			return false, nil // Header doesn't exist
		}
	case "attachment":
		// Check attachment filenames
		var attachmentNames []string
		for _, att := range email.Attachments {
			attachmentNames = append(attachmentNames, att.Filename)
		}
		text = strings.Join(attachmentNames, " ")
	default:
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	// Apply case sensitivity
	searchText := text
	searchValue := condition.Value
	if !condition.CaseSensitive && cr.rulesConfig != nil && !cr.rulesConfig.Settings.CaseSensitive {
		searchText = strings.ToLower(text)
		searchValue = strings.ToLower(condition.Value)
	}

	// Evaluate based on operator
	switch strings.ToLower(condition.Operator) {
	case "contains":
		return strings.Contains(searchText, searchValue), nil
	case "equals":
		return searchText == searchValue, nil
	case "matches", "regex":
		regex, err := regexp.Compile(searchValue)
		if err != nil {
			return false, fmt.Errorf("invalid regex pattern '%s': %v", searchValue, err)
		}
		return regex.MatchString(searchText), nil
	case "starts_with":
		return strings.HasPrefix(searchText, searchValue), nil
	case "ends_with":
		return strings.HasSuffix(searchText, searchValue), nil
	case "length_gt":
		threshold, err := strconv.Atoi(searchValue)
		if err != nil {
			return false, fmt.Errorf("length_gt requires numeric value: %v", err)
		}
		return len(text) > threshold, nil
	case "length_lt":
		threshold, err := strconv.Atoi(searchValue)
		if err != nil {
			return false, fmt.Errorf("length_lt requires numeric value: %v", err)
		}
		return len(text) < threshold, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

// executeRuleActions performs actions when a rule matches
func (cr *CustomRulesPlugin) executeRuleActions(rule Rule, result *PluginResult) {
	// Initialize details if needed
	if result.Details["matched_rules"] == nil {
		result.Details["matched_rules"] = []string{}
	}
	if result.Details["log_messages"] == nil {
		result.Details["log_messages"] = []string{}
	}

	for _, action := range rule.Actions {
		switch strings.ToLower(action.Type) {
		case "score":
			// Score is already added to result.Score
		case "tag":
			// Add tag to metadata
			if result.Metadata["tags"] == "" {
				result.Metadata["tags"] = action.Value
			} else {
				result.Metadata["tags"] += "," + action.Value
			}
		case "log":
			// Add to details for logging
			logMessages := result.Details["log_messages"].([]string)
			result.Details["log_messages"] = append(logMessages, action.Value)
		case "block":
			// Mark as blocked
			result.Details["blocked"] = true
			result.Details["block_reason"] = action.Value
		}
	}
}

// GetRuleStats returns statistics about rule usage
func (cr *CustomRulesPlugin) GetRuleStats() map[string]int {
	statsCopy := make(map[string]int)
	for k, v := range cr.stats {
		statsCopy[k] = v
	}
	return statsCopy
}

// GetLoadedRules returns the currently loaded rules
func (cr *CustomRulesPlugin) GetLoadedRules() []Rule {
	rulesCopy := make([]Rule, len(cr.rules))
	copy(rulesCopy, cr.rules)
	return rulesCopy
}

// GetRulesFile returns the path to the rules file
func (cr *CustomRulesPlugin) GetRulesFile() string {
	return cr.rulesFile
}

// SetRulesFile sets the path to the rules file
func (cr *CustomRulesPlugin) SetRulesFile(path string) {
	cr.rulesFile = path
}
