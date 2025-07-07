package plugins

import (
	"fmt"
	"sync"
)

// DefaultRegistry implements PluginRegistry interface
type DefaultRegistry struct {
	plugins     map[string]Plugin
	pluginTypes map[string][]Plugin // Type name -> plugins
	mu          sync.RWMutex
}

// NewDefaultRegistry creates a new plugin registry
func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		plugins:     make(map[string]Plugin),
		pluginTypes: make(map[string][]Plugin),
	}
}

// Register adds a plugin to the registry
func (r *DefaultRegistry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.plugins[name] = plugin

	// Check which interfaces this plugin implements
	interfaces := []string{
		"ContentAnalyzer",
		"ReputationChecker",
		"AttachmentScanner",
		"MLClassifier",
		"ExternalEngine",
		"CustomRuleEngine",
	}

	for _, interfaceName := range interfaces {
		if r.implementsInterface(plugin, interfaceName) {
			r.pluginTypes[interfaceName] = append(r.pluginTypes[interfaceName], plugin)
		}
	}

	return nil
}

// Get retrieves a plugin by name
func (r *DefaultRegistry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// List returns all registered plugins
func (r *DefaultRegistry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// GetByType returns plugins implementing specific interface
func (r *DefaultRegistry) GetByType(pluginType string) ([]Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins, exists := r.pluginTypes[pluginType]
	if !exists {
		return []Plugin{}, nil
	}

	// Return copy to avoid concurrent modification
	result := make([]Plugin, len(plugins))
	copy(result, plugins)
	return result, nil
}

// IsEnabled is not implemented at registry level - handled by manager
func (r *DefaultRegistry) IsEnabled(name string) bool {
	// This is handled by the plugin manager based on configuration
	// Registry only tracks registration, not enabled state
	_, exists := r.plugins[name]
	return exists
}

// implementsInterface checks if a plugin implements a specific interface
func (r *DefaultRegistry) implementsInterface(plugin Plugin, interfaceName string) bool {
	switch interfaceName {
	case "ContentAnalyzer":
		_, ok := plugin.(ContentAnalyzer)
		return ok
	case "ReputationChecker":
		_, ok := plugin.(ReputationChecker)
		return ok
	case "AttachmentScanner":
		_, ok := plugin.(AttachmentScanner)
		return ok
	case "MLClassifier":
		_, ok := plugin.(MLClassifier)
		return ok
	case "ExternalEngine":
		_, ok := plugin.(ExternalEngine)
		return ok
	case "CustomRuleEngine":
		_, ok := plugin.(CustomRuleEngine)
		return ok
	default:
		return false
	}
}

// GetPluginInfo returns detailed information about a plugin
func (r *DefaultRegistry) GetPluginInfo(name string) (PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return PluginInfo{}, fmt.Errorf("plugin %s not found", name)
	}

	// Determine plugin types
	var types []string
	interfaces := []string{
		"ContentAnalyzer",
		"ReputationChecker",
		"AttachmentScanner",
		"MLClassifier",
		"ExternalEngine",
		"CustomRuleEngine",
	}

	for _, interfaceName := range interfaces {
		if r.implementsInterface(plugin, interfaceName) {
			types = append(types, interfaceName)
		}
	}

	return PluginInfo{
		Name:        plugin.Name(),
		Version:     plugin.Version(),
		Description: plugin.Description(),
		Types:       types,
	}, nil
}

// PluginInfo contains detailed information about a plugin
type PluginInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Types       []string `json:"types"`
}

// ListPluginsByCapability returns plugins that have specific capabilities
func (r *DefaultRegistry) ListPluginsByCapability(capability string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Plugin

	switch capability {
	case "content_analysis":
		result = append(result, r.pluginTypes["ContentAnalyzer"]...)
	case "reputation_check":
		result = append(result, r.pluginTypes["ReputationChecker"]...)
	case "attachment_scan":
		result = append(result, r.pluginTypes["AttachmentScanner"]...)
	case "ml_classification":
		result = append(result, r.pluginTypes["MLClassifier"]...)
	case "external_engine":
		result = append(result, r.pluginTypes["ExternalEngine"]...)
	case "custom_rules":
		result = append(result, r.pluginTypes["CustomRuleEngine"]...)
	}

	return result
}

// Unregister removes a plugin from the registry
func (r *DefaultRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Remove from plugins map
	delete(r.plugins, name)

	// Remove from type maps
	interfaces := []string{
		"ContentAnalyzer",
		"ReputationChecker",
		"AttachmentScanner",
		"MLClassifier",
		"ExternalEngine",
		"CustomRuleEngine",
	}

	for _, interfaceName := range interfaces {
		if plugins, exists := r.pluginTypes[interfaceName]; exists {
			// Remove plugin from slice
			for i, p := range plugins {
				if p.Name() == name {
					r.pluginTypes[interfaceName] = append(plugins[:i], plugins[i+1:]...)
					break
				}
			}
		}
	}

	return nil
}

// Clear removes all plugins from the registry
func (r *DefaultRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]Plugin)
	r.pluginTypes = make(map[string][]Plugin)
}
