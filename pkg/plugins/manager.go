package plugins

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zpam/spam-filter/pkg/email"
)

// DefaultPluginManager implements PluginManager interface
type DefaultPluginManager struct {
	registry     PluginRegistry
	plugins      map[string]Plugin
	configs      map[string]*PluginConfig
	stats        map[string]*internalPluginStats
	aggregation  *ScoreAggregation
	eventHandler EventHandler
	mu           sync.RWMutex
	statsLock    sync.RWMutex
}

// internalPluginStats tracks detailed plugin statistics
type internalPluginStats struct {
	Name           string
	ExecutionCount int64
	TotalTime      time.Duration
	ErrorCount     int64
	LastExecution  time.Time
	mu             sync.RWMutex
}

// NewPluginManager creates a new plugin manager with default settings
func NewPluginManager() *DefaultPluginManager {
	return &DefaultPluginManager{
		registry: NewDefaultRegistry(),
		plugins:  make(map[string]Plugin),
		configs:  make(map[string]*PluginConfig),
		stats:    make(map[string]*internalPluginStats),
		aggregation: &ScoreAggregation{
			Method:    "weighted_sum",
			Weights:   make(map[string]float64),
			Threshold: 35.0, // SpamAssassin-inspired threshold
		},
	}
}

// SetEventHandler sets the event handler for plugin events
func (pm *DefaultPluginManager) SetEventHandler(handler EventHandler) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.eventHandler = handler
}

// SetScoreAggregation configures how plugin scores are combined
func (pm *DefaultPluginManager) SetScoreAggregation(aggregation *ScoreAggregation) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.aggregation = aggregation
}

// RegisterPlugin adds a plugin to the manager
func (pm *DefaultPluginManager) RegisterPlugin(plugin Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Register in registry
	if err := pm.registry.Register(plugin); err != nil {
		return fmt.Errorf("failed to register plugin %s: %v", name, err)
	}

	pm.plugins[name] = plugin
	pm.stats[name] = &internalPluginStats{
		Name: name,
	}

	pm.emitEvent(&PluginEvent{
		Type:      "registered",
		Plugin:    name,
		Timestamp: time.Now(),
	})

	return nil
}

// LoadPlugins loads and initializes all configured plugins
func (pm *DefaultPluginManager) LoadPlugins(configs map[string]*PluginConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.configs = configs

	// Initialize enabled plugins in priority order
	type pluginWithPriority struct {
		name   string
		plugin Plugin
		config *PluginConfig
	}

	var prioritizedPlugins []pluginWithPriority
	for name, config := range configs {
		if !config.Enabled {
			continue
		}

		plugin, exists := pm.plugins[name]
		if !exists {
			return fmt.Errorf("plugin %s not found", name)
		}

		prioritizedPlugins = append(prioritizedPlugins, pluginWithPriority{
			name:   name,
			plugin: plugin,
			config: config,
		})
	}

	// Sort by priority (lower numbers first)
	sort.Slice(prioritizedPlugins, func(i, j int) bool {
		return prioritizedPlugins[i].config.Priority < prioritizedPlugins[j].config.Priority
	})

	// Initialize plugins
	for _, pp := range prioritizedPlugins {
		if err := pp.plugin.Initialize(pp.config); err != nil {
			pm.emitEvent(&PluginEvent{
				Type:      "error",
				Plugin:    pp.name,
				Timestamp: time.Now(),
				Error:     err,
			})
			return fmt.Errorf("failed to initialize plugin %s: %v", pp.name, err)
		}

		pm.emitEvent(&PluginEvent{
			Type:      "loaded",
			Plugin:    pp.name,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// ExecuteAll runs all enabled plugins on an email in parallel
func (pm *DefaultPluginManager) ExecuteAll(ctx context.Context, email *email.Email) ([]*PluginResult, error) {
	pm.mu.RLock()
	enabledPlugins := make([]string, 0, len(pm.configs))
	for name, config := range pm.configs {
		if config.Enabled {
			enabledPlugins = append(enabledPlugins, name)
		}
	}
	pm.mu.RUnlock()

	return pm.executePlugins(ctx, email, enabledPlugins)
}

// ExecuteByType runs plugins of specific type
func (pm *DefaultPluginManager) ExecuteByType(ctx context.Context, email *email.Email, pluginType string) ([]*PluginResult, error) {
	pluginsByType, err := pm.registry.GetByType(pluginType)
	if err != nil {
		return nil, err
	}

	var pluginNames []string
	for _, plugin := range pluginsByType {
		name := plugin.Name()
		if pm.isPluginEnabled(name) {
			pluginNames = append(pluginNames, name)
		}
	}

	return pm.executePlugins(ctx, email, pluginNames)
}

// executePlugins executes specified plugins in parallel
func (pm *DefaultPluginManager) executePlugins(ctx context.Context, email *email.Email, pluginNames []string) ([]*PluginResult, error) {
	if len(pluginNames) == 0 {
		return []*PluginResult{}, nil
	}

	// Create channels for parallel execution
	resultChan := make(chan *PluginResult, len(pluginNames))
	var wg sync.WaitGroup

	// Execute plugins in parallel
	for _, name := range pluginNames {
		wg.Add(1)
		go func(pluginName string) {
			defer wg.Done()
			result := pm.executePlugin(ctx, pluginName, email)
			resultChan <- result
		}(name)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []*PluginResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results, nil
}

// executePlugin executes a single plugin with timeout and error handling
func (pm *DefaultPluginManager) executePlugin(ctx context.Context, name string, email *email.Email) *PluginResult {
	start := time.Now()

	pm.mu.RLock()
	plugin, exists := pm.plugins[name]
	config := pm.configs[name]
	pm.mu.RUnlock()

	result := &PluginResult{
		Name:        name,
		ProcessTime: 0,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
	}

	if !exists {
		result.Error = fmt.Errorf("plugin %s not found", name)
		return result
	}

	// Create timeout context
	timeout := 5 * time.Second // Default timeout
	if config != nil && config.Timeout > 0 {
		timeout = config.Timeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute plugin based on its type
	var err error
	switch p := plugin.(type) {
	case ContentAnalyzer:
		result, err = p.AnalyzeContent(timeoutCtx, email)
	case ReputationChecker:
		result, err = p.CheckReputation(timeoutCtx, email)
	case AttachmentScanner:
		result, err = p.ScanAttachments(timeoutCtx, email.Attachments)
	case MLClassifier:
		result, err = p.Classify(timeoutCtx, email)
	case ExternalEngine:
		result, err = p.Analyze(timeoutCtx, email)
	case CustomRuleEngine:
		result, err = p.EvaluateRules(timeoutCtx, email)
	default:
		result.Error = fmt.Errorf("unknown plugin type for %s", name)
	}

	processingTime := time.Since(start)

	if err != nil {
		result = &PluginResult{
			Name:        name,
			Score:       0,
			Confidence:  0,
			ProcessTime: processingTime,
			Error:       err,
			Details:     make(map[string]any),
			Metadata:    make(map[string]string),
		}
	} else if result != nil {
		// Ensure result has correct timing and name
		result.Name = name
		result.ProcessTime = processingTime

		// Apply plugin weight if configured
		if config != nil && config.Weight != 0 {
			result.Score *= config.Weight
		}
	}

	// Update statistics
	pm.updateStats(name, processingTime, err != nil)

	// Emit event
	eventType := "executed"
	if err != nil {
		eventType = "error"
	}

	pm.emitEvent(&PluginEvent{
		Type:      eventType,
		Plugin:    name,
		Timestamp: time.Now(),
		Data:      result,
		Error:     err,
	})

	return result
}

// CombineScores aggregates plugin results into final score
func (pm *DefaultPluginManager) CombineScores(results []*PluginResult) (float64, error) {
	pm.mu.RLock()
	aggregation := pm.aggregation
	pm.mu.RUnlock()

	if aggregation == nil {
		return 0, fmt.Errorf("no score aggregation configured")
	}

	switch aggregation.Method {
	case "weighted_sum":
		return pm.weightedSum(results, aggregation.Weights)
	case "max":
		return pm.maxScore(results)
	case "average":
		return pm.averageScore(results)
	case "consensus":
		return pm.consensusScore(results, aggregation.Threshold)
	default:
		return 0, fmt.Errorf("unknown aggregation method: %s", aggregation.Method)
	}
}

// weightedSum calculates weighted sum of plugin scores
func (pm *DefaultPluginManager) weightedSum(results []*PluginResult, weights map[string]float64) (float64, error) {
	var totalScore float64
	var totalWeight float64

	for _, result := range results {
		if result.Error != nil {
			continue // Skip failed plugins
		}

		weight := 1.0 // Default weight
		if w, exists := weights[result.Name]; exists {
			weight = w
		}

		totalScore += result.Score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0, nil
	}

	// Return normalized score
	return totalScore, nil
}

// maxScore returns the highest score from all plugins
func (pm *DefaultPluginManager) maxScore(results []*PluginResult) (float64, error) {
	var maxScore float64
	found := false

	for _, result := range results {
		if result.Error != nil {
			continue
		}
		if !found || result.Score > maxScore {
			maxScore = result.Score
			found = true
		}
	}

	return maxScore, nil
}

// averageScore returns the average score from all plugins
func (pm *DefaultPluginManager) averageScore(results []*PluginResult) (float64, error) {
	var totalScore float64
	count := 0

	for _, result := range results {
		if result.Error != nil {
			continue
		}
		totalScore += result.Score
		count++
	}

	if count == 0 {
		return 0, nil
	}

	return totalScore / float64(count), nil
}

// consensusScore returns spam score based on plugin consensus
func (pm *DefaultPluginManager) consensusScore(results []*PluginResult, threshold float64) (float64, error) {
	spamVotes := 0
	totalVotes := 0

	for _, result := range results {
		if result.Error != nil {
			continue
		}

		totalVotes++
		if result.Score >= threshold {
			spamVotes++
		}
	}

	if totalVotes == 0 {
		return 0, nil
	}

	// Return score based on consensus percentage
	consensus := float64(spamVotes) / float64(totalVotes)
	if consensus >= 0.5 {
		return threshold + (consensus-0.5)*20, nil // Scale above threshold
	}

	return consensus * threshold, nil
}

// GetStats returns plugin execution statistics
func (pm *DefaultPluginManager) GetStats() map[string]PluginStats {
	pm.statsLock.RLock()
	defer pm.statsLock.RUnlock()

	stats := make(map[string]PluginStats)
	for name, internal := range pm.stats {
		internal.mu.RLock()

		var avgTime time.Duration
		var successRate float64

		if internal.ExecutionCount > 0 {
			avgTime = time.Duration(internal.TotalTime.Nanoseconds() / internal.ExecutionCount)
			successRate = float64(internal.ExecutionCount-internal.ErrorCount) / float64(internal.ExecutionCount)
		}

		stats[name] = PluginStats{
			Name:           internal.Name,
			ExecutionCount: internal.ExecutionCount,
			TotalTime:      internal.TotalTime,
			AverageTime:    avgTime,
			ErrorCount:     internal.ErrorCount,
			LastExecution:  internal.LastExecution,
			SuccessRate:    successRate,
		}

		internal.mu.RUnlock()
	}

	return stats
}

// updateStats updates plugin execution statistics
func (pm *DefaultPluginManager) updateStats(name string, duration time.Duration, hasError bool) {
	pm.statsLock.RLock()
	stats, exists := pm.stats[name]
	pm.statsLock.RUnlock()

	if !exists {
		return
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	atomic.AddInt64(&stats.ExecutionCount, 1)
	stats.TotalTime += duration
	stats.LastExecution = time.Now()

	if hasError {
		atomic.AddInt64(&stats.ErrorCount, 1)
	}
}

// isPluginEnabled checks if a plugin is enabled
func (pm *DefaultPluginManager) isPluginEnabled(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	config, exists := pm.configs[name]
	return exists && config.Enabled
}

// emitEvent sends plugin events to handler
func (pm *DefaultPluginManager) emitEvent(event *PluginEvent) {
	if pm.eventHandler != nil {
		pm.eventHandler.HandleEvent(event)
	}
}

// Shutdown gracefully shuts down all plugins
func (pm *DefaultPluginManager) Shutdown(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var shutdownErrors []error

	for name, plugin := range pm.plugins {
		if err := plugin.Cleanup(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("plugin %s cleanup failed: %v", name, err))
		}

		pm.emitEvent(&PluginEvent{
			Type:      "unloaded",
			Plugin:    name,
			Timestamp: time.Now(),
		})
	}

	if len(shutdownErrors) > 0 {
		return fmt.Errorf("shutdown errors: %v", shutdownErrors)
	}

	return nil
}
