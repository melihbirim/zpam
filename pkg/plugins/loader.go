package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginLoader handles loading plugins from various sources
type PluginLoader struct {
	registry   PluginRegistry
	pluginsDir string
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(registry PluginRegistry) *PluginLoader {
	return &PluginLoader{
		registry:   registry,
		pluginsDir: "plugins", // Default plugins directory
	}
}

// SetPluginsDirectory sets the directory where plugins are located
func (pl *PluginLoader) SetPluginsDirectory(dir string) {
	pl.pluginsDir = dir
}

// LoadFromDirectory scans a directory and loads all valid plugins
func (pl *PluginLoader) LoadFromDirectory() error {
	if _, err := os.Stat(pl.pluginsDir); os.IsNotExist(err) {
		// Plugins directory doesn't exist, skip loading
		return nil
	}

	// Walk through plugins directory
	return filepath.Walk(pl.pluginsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for Lua plugins
		if strings.HasSuffix(path, "main.lua") {
			// Look for manifest in parent directory
			pluginDir := filepath.Dir(path)
			manifestPath := filepath.Join(pluginDir, "zpam-plugin.yaml")
			if _, err := os.Stat(manifestPath); err == nil {
				return pl.loadLuaPlugin(pluginDir)
			}
		}

		return nil
	})
}

// loadLuaPlugin loads a Lua plugin from a directory
func (pl *PluginLoader) loadLuaPlugin(pluginDir string) error {
	// Read manifest
	manifestPath := filepath.Join(pluginDir, "zpam-plugin.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest %s: %v", manifestPath, err)
	}

	var manifest LuaPluginManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest %s: %v", manifestPath, err)
	}

	// Check if this is a Lua plugin
	if manifest.Main.Runtime != "lua" {
		return nil // Not a Lua plugin
	}

	// Create Lua plugin from script
	scriptPath := filepath.Join(pluginDir, manifest.Main.Script)
	luaPlugin, err := NewLuaPlugin(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create Lua plugin from %s: %v", scriptPath, err)
	}

	// Set plugin metadata from manifest
	luaPlugin.name = manifest.Plugin.Name
	luaPlugin.version = manifest.Plugin.Version
	luaPlugin.description = manifest.Plugin.Description
	luaPlugin.pluginType = manifest.Plugin.Type
	luaPlugin.interfaces = manifest.Interfaces

	// Register the plugin
	if err := pl.registry.Register(luaPlugin); err != nil {
		return fmt.Errorf("failed to register Lua plugin %s: %v", luaPlugin.name, err)
	}

	return nil
}

// LuaPluginManifest represents the YAML manifest for Lua plugins
type LuaPluginManifest struct {
	ManifestVersion string `yaml:"manifest_version"`
	Plugin          struct {
		Name        string   `yaml:"name"`
		Version     string   `yaml:"version"`
		Description string   `yaml:"description"`
		Author      string   `yaml:"author"`
		Homepage    string   `yaml:"homepage"`
		Repository  string   `yaml:"repository"`
		License     string   `yaml:"license"`
		Type        string   `yaml:"type"`
		Tags        []string `yaml:"tags"`
		MinVersion  string   `yaml:"min_zpam_version"`
	} `yaml:"plugin"`
	Main struct {
		Script  string `yaml:"script"`
		Runtime string `yaml:"runtime"`
	} `yaml:"main"`
	Interfaces []string `yaml:"interfaces"`
	Security   struct {
		Permissions []string `yaml:"permissions"`
		Sandbox     bool     `yaml:"sandbox"`
	} `yaml:"security"`
	Configuration map[string]interface{} `yaml:"configuration"`
}

// LoadBuiltinPlugins registers built-in Go plugins
func (pl *PluginLoader) LoadBuiltinPlugins() error {
	// Register built-in plugins
	builtinPlugins := []Plugin{
		NewSpamAssassinPlugin(),
		NewRspamdPlugin(),
		NewCustomRulesPlugin(),
		NewMLPlugin(),
		NewVirusTotalPlugin(),
	}

	for _, plugin := range builtinPlugins {
		if err := pl.registry.Register(plugin); err != nil {
			return fmt.Errorf("failed to register builtin plugin %s: %v", plugin.Name(), err)
		}
	}

	return nil
}

// DiscoverPlugins discovers and loads all available plugins
func (pl *PluginLoader) DiscoverPlugins() error {
	// First load built-in Go plugins
	if err := pl.LoadBuiltinPlugins(); err != nil {
		return fmt.Errorf("failed to load builtin plugins: %v", err)
	}

	// Then load Lua plugins from directory
	if err := pl.LoadFromDirectory(); err != nil {
		return fmt.Errorf("failed to load plugins from directory: %v", err)
	}

	return nil
}
