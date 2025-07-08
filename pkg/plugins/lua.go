package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
	"github.com/zpam/spam-filter/pkg/email"
)

// LuaPlugin represents a plugin written in Lua
type LuaPlugin struct {
	name        string
	version     string
	description string
	scriptPath  string
	config      *PluginConfig
	enabled     bool

	// Plugin type information
	pluginType string   // "content-analyzer", "reputation-checker", etc.
	interfaces []string // Which interfaces this plugin implements

	// Lua VM pool for concurrent execution
	vmPool chan *lua.LState
	maxVMs int
}

// NewLuaPlugin creates a new Lua plugin from a script file
func NewLuaPlugin(scriptPath string) (*LuaPlugin, error) {
	// Read the script to extract metadata
	metadata, err := extractLuaMetadata(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %v", err)
	}

	plugin := &LuaPlugin{
		name:        metadata.Name,
		version:     metadata.Version,
		description: metadata.Description,
		scriptPath:  scriptPath,
		enabled:     false,
		pluginType:  metadata.Type,
		interfaces:  metadata.Interfaces,
		maxVMs:      5, // Pool of 5 VMs for concurrent execution
	}

	// Initialize VM pool
	plugin.vmPool = make(chan *lua.LState, plugin.maxVMs)

	return plugin, nil
}

// LuaPluginMetadata represents metadata extracted from Lua script comments
type LuaPluginMetadata struct {
	Name        string
	Version     string
	Description string
	Type        string
	Interfaces  []string
}

// extractLuaMetadata extracts plugin metadata from Lua script comments
func extractLuaMetadata(scriptPath string) (*LuaPluginMetadata, error) {
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	metadata := &LuaPluginMetadata{
		Name:        filepath.Base(scriptPath),
		Version:     "1.0.0",
		Description: "Lua plugin",
		Type:        "content-analyzer",
		Interfaces:  []string{"ContentAnalyzer"},
	}

	// Parse metadata from comments at the top of the file
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "--") {
			break // Stop at first non-comment line
		}

		comment := strings.TrimPrefix(line, "--")
		comment = strings.TrimSpace(comment)

		// Parse metadata fields
		if strings.HasPrefix(comment, "@name") {
			metadata.Name = strings.TrimSpace(strings.TrimPrefix(comment, "@name"))
		} else if strings.HasPrefix(comment, "@version") {
			metadata.Version = strings.TrimSpace(strings.TrimPrefix(comment, "@version"))
		} else if strings.HasPrefix(comment, "@description") {
			metadata.Description = strings.TrimSpace(strings.TrimPrefix(comment, "@description"))
		} else if strings.HasPrefix(comment, "@type") {
			metadata.Type = strings.TrimSpace(strings.TrimPrefix(comment, "@type"))
		} else if strings.HasPrefix(comment, "@interfaces") {
			interfaceList := strings.TrimSpace(strings.TrimPrefix(comment, "@interfaces"))
			metadata.Interfaces = strings.Split(interfaceList, ",")
			for i, iface := range metadata.Interfaces {
				metadata.Interfaces[i] = strings.TrimSpace(iface)
			}
		}
	}

	return metadata, nil
}

// Plugin interface implementation
func (lp *LuaPlugin) Name() string {
	return lp.name
}

func (lp *LuaPlugin) Version() string {
	return lp.version
}

func (lp *LuaPlugin) Description() string {
	return lp.description
}

func (lp *LuaPlugin) Initialize(config *PluginConfig) error {
	lp.config = config
	lp.enabled = config.Enabled

	if !lp.enabled {
		return nil
	}

	// Pre-create VM pool
	for i := 0; i < lp.maxVMs; i++ {
		vm, err := lp.createVM()
		if err != nil {
			return fmt.Errorf("failed to create Lua VM: %v", err)
		}
		lp.vmPool <- vm
	}

	return nil
}

func (lp *LuaPlugin) IsHealthy(ctx context.Context) error {
	if !lp.enabled {
		return fmt.Errorf("lua plugin not enabled")
	}

	// Test script execution with a simple test
	vm := lp.getVM()
	defer lp.returnVM(vm)

	if err := vm.DoString("return true"); err != nil {
		return fmt.Errorf("lua execution failed: %v", err)
	}

	return nil
}

func (lp *LuaPlugin) Cleanup() error {
	// Close all VMs in the pool
	close(lp.vmPool)
	for vm := range lp.vmPool {
		vm.Close()
	}
	return nil
}

// createVM creates a new Lua VM with the plugin script loaded
func (lp *LuaPlugin) createVM() (*lua.LState, error) {
	vm := lua.NewState()

	// Register ZPAM API functions
	lp.registerZPAMAPI(vm)

	// Load the plugin script
	if err := vm.DoFile(lp.scriptPath); err != nil {
		vm.Close()
		return nil, fmt.Errorf("failed to load script %s: %v", lp.scriptPath, err)
	}

	return vm, nil
}

// getVM gets a VM from the pool or creates a new one
func (lp *LuaPlugin) getVM() *lua.LState {
	select {
	case vm := <-lp.vmPool:
		return vm
	default:
		// Pool is empty, create a new VM
		vm, err := lp.createVM()
		if err != nil {
			// Fallback: return a basic VM
			return lua.NewState()
		}
		return vm
	}
}

// returnVM returns a VM to the pool
func (lp *LuaPlugin) returnVM(vm *lua.LState) {
	select {
	case lp.vmPool <- vm:
		// Successfully returned to pool
	default:
		// Pool is full, close the VM
		vm.Close()
	}
}

// registerZPAMAPI registers ZPAM-specific functions for Lua scripts
func (lp *LuaPlugin) registerZPAMAPI(vm *lua.LState) {
	// Create zpam global table
	zpamTable := vm.NewTable()
	vm.SetGlobal("zpam", zpamTable)

	// Register utility functions
	vm.SetField(zpamTable, "log", vm.NewFunction(lp.luaLog))
	vm.SetField(zpamTable, "contains", vm.NewFunction(lp.luaContains))
	vm.SetField(zpamTable, "regex_match", vm.NewFunction(lp.luaRegexMatch))
	vm.SetField(zpamTable, "domain_from_email", vm.NewFunction(lp.luaDomainFromEmail))
}

// Lua API functions
func (lp *LuaPlugin) luaLog(vm *lua.LState) int {
	message := vm.CheckString(1)
	fmt.Printf("[Lua Plugin %s] %s\n", lp.name, message)
	return 0
}

func (lp *LuaPlugin) luaContains(vm *lua.LState) int {
	haystack := vm.CheckString(1)
	needle := vm.CheckString(2)
	result := strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
	vm.Push(lua.LBool(result))
	return 1
}

func (lp *LuaPlugin) luaRegexMatch(vm *lua.LState) int {
	// TODO: Implement regex matching
	vm.Push(lua.LBool(false))
	return 1
}

func (lp *LuaPlugin) luaDomainFromEmail(vm *lua.LState) int {
	emailAddr := vm.CheckString(1)
	parts := strings.Split(emailAddr, "@")
	if len(parts) == 2 {
		vm.Push(lua.LString(parts[1]))
	} else {
		vm.Push(lua.LString(""))
	}
	return 1
}

// convertEmailToLua converts an email.Email to Lua table
func (lp *LuaPlugin) convertEmailToLua(vm *lua.LState, email *email.Email) *lua.LTable {
	emailTable := vm.NewTable()

	// Basic fields
	vm.SetField(emailTable, "from", lua.LString(email.From))
	vm.SetField(emailTable, "to", lua.LString(strings.Join(email.To, ",")))
	vm.SetField(emailTable, "subject", lua.LString(email.Subject))
	vm.SetField(emailTable, "body", lua.LString(email.Body))

	// Headers
	headersTable := vm.NewTable()
	for key, value := range email.Headers {
		vm.SetField(headersTable, key, lua.LString(value))
	}
	vm.SetField(emailTable, "headers", headersTable)

	// Attachments
	attachmentsTable := vm.NewTable()
	for i, attachment := range email.Attachments {
		attTable := vm.NewTable()
		vm.SetField(attTable, "filename", lua.LString(attachment.Filename))
		vm.SetField(attTable, "content_type", lua.LString(attachment.ContentType))
		vm.SetField(attTable, "size", lua.LNumber(attachment.Size))
		vm.RawSetInt(attachmentsTable, i+1, attTable)
	}
	vm.SetField(emailTable, "attachments", attachmentsTable)

	return emailTable
}

// convertLuaToResult converts Lua table to PluginResult
func (lp *LuaPlugin) convertLuaToResult(vm *lua.LState, luaResult lua.LValue) (*PluginResult, error) {
	if luaResult.Type() != lua.LTTable {
		return nil, fmt.Errorf("plugin must return a table")
	}

	resultTable := luaResult.(*lua.LTable)

	result := &PluginResult{
		Name:        lp.name,
		Score:       0,
		Confidence:  0.7,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
		Rules:       []string{},
		ProcessTime: 0,
	}

	// Extract score
	if scoreVal := vm.GetField(resultTable, "score"); scoreVal.Type() == lua.LTNumber {
		result.Score = float64(scoreVal.(lua.LNumber))
	}

	// Extract confidence
	if confVal := vm.GetField(resultTable, "confidence"); confVal.Type() == lua.LTNumber {
		result.Confidence = float64(confVal.(lua.LNumber))
	}

	// Extract rules
	if rulesVal := vm.GetField(resultTable, "rules"); rulesVal.Type() == lua.LTTable {
		rulesTable := rulesVal.(*lua.LTable)
		rulesTable.ForEach(func(_, value lua.LValue) {
			if value.Type() == lua.LTString {
				result.Rules = append(result.Rules, string(value.(lua.LString)))
			}
		})
	}

	// Extract metadata
	if metaVal := vm.GetField(resultTable, "metadata"); metaVal.Type() == lua.LTTable {
		metaTable := metaVal.(*lua.LTable)
		metaTable.ForEach(func(key, value lua.LValue) {
			if key.Type() == lua.LTString && value.Type() == lua.LTString {
				result.Metadata[string(key.(lua.LString))] = string(value.(lua.LString))
			}
		})
	}

	return result, nil
}

// Interface implementations - these check if the Lua script implements the required functions

// ContentAnalyzer implementation
func (lp *LuaPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return lp.executeFunction(ctx, "analyze_content", email)
}

// ReputationChecker implementation
func (lp *LuaPlugin) CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return lp.executeFunction(ctx, "check_reputation", email)
}

// AttachmentScanner implementation
func (lp *LuaPlugin) ScanAttachments(ctx context.Context, attachments []email.Attachment) (*PluginResult, error) {
	// Create a fake email with just attachments for consistency
	fakeEmail := &email.Email{Attachments: attachments}
	return lp.executeFunction(ctx, "scan_attachments", fakeEmail)
}

// MLClassifier implementation
func (lp *LuaPlugin) Classify(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return lp.executeFunction(ctx, "classify", email)
}

func (lp *LuaPlugin) Train(ctx context.Context, emails []email.Email, labels []bool) error {
	// Training not supported for Lua plugins yet
	return fmt.Errorf("training not supported for Lua plugins")
}

// ExternalEngine implementation
func (lp *LuaPlugin) Analyze(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return lp.executeFunction(ctx, "analyze", email)
}

func (lp *LuaPlugin) GetEngineStats(ctx context.Context) (map[string]any, error) {
	return map[string]any{}, nil
}

// CustomRuleEngine implementation
func (lp *LuaPlugin) EvaluateRules(ctx context.Context, email *email.Email) (*PluginResult, error) {
	return lp.executeFunction(ctx, "evaluate_rules", email)
}

func (lp *LuaPlugin) LoadRules(rules []Rule) error {
	// Rules loading not supported for Lua plugins yet
	return nil
}

// executeFunction executes a Lua function with email data
func (lp *LuaPlugin) executeFunction(ctx context.Context, functionName string, email *email.Email) (*PluginResult, error) {
	start := time.Now()

	if !lp.enabled {
		return &PluginResult{
			Name:        lp.name,
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("plugin not enabled"),
		}, nil
	}

	vm := lp.getVM()
	defer lp.returnVM(vm)

	// Check if function exists
	fnValue := vm.GetGlobal(functionName)
	if fnValue.Type() != lua.LTFunction {
		return &PluginResult{
			Name:        lp.name,
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("function %s not found in script", functionName),
		}, nil
	}

	// Convert email to Lua table
	emailTable := lp.convertEmailToLua(vm, email)

	// Create timeout context for Lua execution
	timeout := 5 * time.Second
	if lp.config != nil && lp.config.Timeout > 0 {
		timeout = lp.config.Timeout
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute function with timeout
	done := make(chan struct{})
	var luaResult lua.LValue
	var luaErr error

	go func() {
		defer close(done)
		vm.Push(fnValue)
		vm.Push(emailTable)
		luaErr = vm.PCall(1, 1, nil)
		if luaErr == nil {
			luaResult = vm.Get(-1)
			vm.Pop(1)
		}
	}()

	select {
	case <-timeoutCtx.Done():
		return &PluginResult{
			Name:        lp.name,
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("lua execution timeout"),
		}, nil
	case <-done:
		if luaErr != nil {
			return &PluginResult{
				Name:        lp.name,
				Score:       0,
				Confidence:  0,
				ProcessTime: time.Since(start),
				Error:       fmt.Errorf("lua execution error: %v", luaErr),
			}, nil
		}
	}

	// Convert result
	result, err := lp.convertLuaToResult(vm, luaResult)
	if err != nil {
		return &PluginResult{
			Name:        lp.name,
			Score:       0,
			Confidence:  0,
			ProcessTime: time.Since(start),
			Error:       fmt.Errorf("result conversion error: %v", err),
		}, nil
	}

	result.ProcessTime = time.Since(start)
	return result, nil
}

// GetPluginType returns the plugin type
func (lp *LuaPlugin) GetPluginType() string {
	return lp.pluginType
}

// GetInterfaces returns the interfaces this plugin implements
func (lp *LuaPlugin) GetInterfaces() []string {
	return lp.interfaces
}
