# Custom Plugin Development for ZPO

ZPO provides a powerful plugin system that allows developers to create custom spam detection modules. This guide covers everything you need to know to develop, test, and deploy custom plugins.

## Plugin Architecture

ZPO uses an interface-based plugin system with multiple specialized interfaces:

- **ContentAnalyzer**: Analyze email content for spam indicators
- **ReputationChecker**: Check sender/domain/URL reputation
- **AttachmentScanner**: Scan email attachments for threats
- **MLClassifier**: Machine learning-based classification
- **ExternalEngine**: Integration with external services
- **CustomRuleEngine**: User-defined rule evaluation

## Quick Start

### 1. Create Plugin File

Create your plugin in `pkg/plugins/your_plugin_name.go`:

```go
package plugins

import (
    "context"
    "fmt"
    "time"
    
    "github.com/zpo/spam-filter/pkg/email"
)

type YourPlugin struct {
    config  *PluginConfig
    enabled bool
    // Add your fields here
}

func NewYourPlugin() *YourPlugin {
    return &YourPlugin{}
}

func (yp *YourPlugin) Name() string {
    return "your_plugin"
}

func (yp *YourPlugin) Version() string {
    return "1.0.0"
}

func (yp *YourPlugin) Description() string {
    return "Your custom plugin description"
}

func (yp *YourPlugin) Initialize(config *PluginConfig) error {
    yp.config = config
    yp.enabled = config.Enabled
    // Add initialization logic
    return nil
}

func (yp *YourPlugin) IsHealthy(ctx context.Context) error {
    if !yp.enabled {
        return fmt.Errorf("plugin not enabled")
    }
    // Add health checks
    return nil
}

func (yp *YourPlugin) Cleanup() error {
    // Cleanup resources
    return nil
}
```

### 2. Implement Plugin Interface

Choose one or more interfaces based on your plugin's functionality:

```go
// For content analysis
func (yp *YourPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
    // Implement content analysis
    return &PluginResult{
        Name:       yp.Name(),
        Score:      0.0,
        Confidence: 0.5,
        // ... other fields
    }, nil
}

// For reputation checking
func (yp *YourPlugin) CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error) {
    // Implement reputation checking
    return &PluginResult{
        Name:       yp.Name(),
        Score:      0.0,
        Confidence: 0.7,
        // ... other fields
    }, nil
}
```

### 3. Register Plugin

Add your plugin to `pkg/filter/spam_filter.go`:

```go
sf.pluginManager.RegisterPlugin(plugins.NewYourPlugin())
```

Add to `cmd/plugins.go`:

```go
pluginManager.RegisterPlugin(plugins.NewYourPlugin())

// In the switch statement for test-one command:
case "your_plugin":
    plugin = plugins.NewYourPlugin()
    pluginConfig = convertConfigToPluginConfig(cfg.Plugins.YourPlugin)
```

### 4. Add Configuration

Add to your `config.yaml`:

```yaml
plugins:
  your_plugin:
    enabled: true
    weight: 2.0
    priority: 10
    timeout_ms: 5000
    settings:
      api_key: "your-api-key"
      endpoint: "https://api.example.com"
      custom_setting: "value"
```

## Plugin Interfaces

### ContentAnalyzer Interface

Best for plugins that analyze email text, HTML, or structure:

```go
type ContentAnalyzer interface {
    Plugin
    AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error)
}
```

**Example Use Cases:**
- Keyword detection
- Language analysis
- HTML structure analysis
- Text classification

**Implementation Example:**

```go
func (cp *ContentPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
    result := &PluginResult{
        Name:       cp.Name(),
        Score:      0,
        Confidence: 0.5,
        Details:    make(map[string]any),
        Metadata:   make(map[string]string),
        Rules:      []string{},
    }
    
    // Analyze subject line
    if strings.Contains(strings.ToLower(email.Subject), "urgent") {
        result.Score += 5.0
        result.Rules = append(result.Rules, "Urgent keyword detected")
    }
    
    // Analyze body content
    bodyLength := len(email.Body)
    if bodyLength < 50 {
        result.Score += 3.0
        result.Rules = append(result.Rules, "Very short email body")
    }
    
    result.Metadata["body_length"] = fmt.Sprintf("%d", bodyLength)
    result.Confidence = 0.8
    
    return result, nil
}
```

### ReputationChecker Interface

Best for plugins that check external reputation sources:

```go
type ReputationChecker interface {
    Plugin
    CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error)
}
```

**Example Use Cases:**
- Domain blacklist checking
- IP reputation lookup
- URL reputation analysis
- Sender reputation scoring

**Implementation Example:**

```go
func (rp *ReputationPlugin) CheckReputation(ctx context.Context, email *email.Email) (*PluginResult, error) {
    result := &PluginResult{
        Name:       rp.Name(),
        Score:      0,
        Confidence: 0.7,
        Details:    make(map[string]any),
        Metadata:   make(map[string]string),
        Rules:      []string{},
    }
    
    // Extract domain from sender
    domain := rp.extractDomain(email.From)
    
    // Check domain reputation (example)
    reputation, err := rp.checkDomainReputation(ctx, domain)
    if err != nil {
        return result, err
    }
    
    if reputation.IsBlacklisted {
        result.Score = 50.0
        result.Rules = append(result.Rules, fmt.Sprintf("Domain %s is blacklisted", domain))
        result.Confidence = 0.9
    } else if reputation.Score < 0.3 {
        result.Score = 15.0
        result.Rules = append(result.Rules, fmt.Sprintf("Domain %s has poor reputation", domain))
        result.Confidence = 0.6
    }
    
    result.Metadata["domain"] = domain
    result.Metadata["reputation_score"] = fmt.Sprintf("%.2f", reputation.Score)
    
    return result, nil
}
```

### MLClassifier Interface

Best for machine learning-based plugins:

```go
type MLClassifier interface {
    Plugin
    Classify(ctx context.Context, email *email.Email) (*PluginResult, error)
    Train(ctx context.Context, emails []email.Email, labels []bool) error
}
```

**Example Use Cases:**
- Deep learning models
- Ensemble methods
- Feature-based classification
- Real-time learning systems

### ExternalEngine Interface

Best for integrating with external services:

```go
type ExternalEngine interface {
    Plugin
    Analyze(ctx context.Context, email *email.Email) (*PluginResult, error)
    GetEngineStats(ctx context.Context) (map[string]any, error)
}
```

**Example Use Cases:**
- Commercial spam detection APIs
- Cloud-based analysis services
- Legacy system integration
- Third-party ML services

## Advanced Plugin Development

### Error Handling

Always handle errors gracefully and provide meaningful error messages:

```go
func (cp *CustomPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
    result := &PluginResult{
        Name: cp.Name(),
        // ... initialize fields
    }
    
    defer func() {
        if r := recover(); r != nil {
            result.Error = fmt.Errorf("plugin panic: %v", r)
        }
    }()
    
    // Your analysis logic here
    score, err := cp.performAnalysis(email)
    if err != nil {
        result.Error = fmt.Errorf("analysis failed: %w", err)
        return result, nil // Return result with error, not error
    }
    
    result.Score = score
    return result, nil
}
```

### Configuration Management

Handle configuration robustly with defaults and validation:

```go
func (cp *CustomPlugin) Initialize(config *PluginConfig) error {
    cp.config = config
    cp.enabled = config.Enabled
    
    if !cp.enabled {
        return nil
    }
    
    // Extract settings with defaults
    if config.Settings != nil {
        if apiKey, ok := config.Settings["api_key"].(string); ok {
            cp.apiKey = apiKey
        }
        
        if endpoint, ok := config.Settings["endpoint"].(string); ok {
            cp.endpoint = endpoint
        } else {
            cp.endpoint = "https://api.default.com" // Default
        }
        
        if timeout, ok := config.Settings["timeout"].(float64); ok {
            cp.timeout = time.Duration(timeout) * time.Millisecond
        } else {
            cp.timeout = 10 * time.Second // Default
        }
    }
    
    // Validate required settings
    if cp.apiKey == "" {
        return fmt.Errorf("api_key is required")
    }
    
    return nil
}
```

### Performance Monitoring

Include performance tracking in your plugins:

```go
type CustomPluginStats struct {
    RequestsTotal   int64   `json:"requests_total"`
    RequestsFailed  int64   `json:"requests_failed"`
    AverageResponse float64 `json:"average_response_ms"`
    LastUpdate      string  `json:"last_update"`
}

func (cp *CustomPlugin) updateStats(duration time.Duration, err error) {
    cp.stats.RequestsTotal++
    if err != nil {
        cp.stats.RequestsFailed++
    }
    
    responseTime := float64(duration.Nanoseconds()) / 1e6
    if cp.stats.RequestsTotal == 1 {
        cp.stats.AverageResponse = responseTime
    } else {
        cp.stats.AverageResponse = (cp.stats.AverageResponse*float64(cp.stats.RequestsTotal-1) + responseTime) / float64(cp.stats.RequestsTotal)
    }
    
    cp.stats.LastUpdate = time.Now().Format(time.RFC3339)
}
```

### External API Integration

Example of integrating with external APIs:

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

func (cp *CustomPlugin) callExternalAPI(ctx context.Context, email *email.Email) (*APIResponse, error) {
    // Prepare request
    request := &APIRequest{
        Subject: email.Subject,
        Body:    email.Body,
        From:    email.From,
    }
    
    requestBody, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    // Create HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", cp.endpoint, bytes.NewBuffer(requestBody))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+cp.apiKey)
    
    // Make request
    client := &http.Client{Timeout: cp.timeout}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // Parse response
    var response APIResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &response, nil
}
```

## Testing Your Plugin

### Unit Testing

Create comprehensive unit tests for your plugin:

```go
// pkg/plugins/your_plugin_test.go
package plugins

import (
    "context"
    "testing"
    "time"
    
    "github.com/zpo/spam-filter/pkg/email"
)

func TestYourPlugin_AnalyzeContent(t *testing.T) {
    plugin := NewYourPlugin()
    
    config := &PluginConfig{
        Enabled: true,
        Settings: map[string]any{
            "api_key": "test-key",
        },
    }
    
    err := plugin.Initialize(config)
    if err != nil {
        t.Fatalf("Failed to initialize plugin: %v", err)
    }
    
    testEmail := &email.Email{
        Subject: "URGENT: Win $1000000 NOW!!!",
        Body:    "Click here to claim your prize",
        From:    "spam@example.com",
    }
    
    ctx := context.Background()
    result, err := plugin.AnalyzeContent(ctx, testEmail)
    
    if err != nil {
        t.Fatalf("AnalyzeContent failed: %v", err)
    }
    
    if result.Score <= 0 {
        t.Errorf("Expected positive spam score, got %f", result.Score)
    }
    
    if len(result.Rules) == 0 {
        t.Error("Expected triggered rules, got none")
    }
}

func TestYourPlugin_Configuration(t *testing.T) {
    plugin := NewYourPlugin()
    
    // Test missing required config
    config := &PluginConfig{
        Enabled:  true,
        Settings: map[string]any{},
    }
    
    err := plugin.Initialize(config)
    if err == nil {
        t.Error("Expected error for missing api_key, got nil")
    }
}
```

### Integration Testing

Test your plugin with real emails:

```bash
# Create test email
echo 'Subject: Test Email
From: test@example.com

This is a test email body.' > test_email.eml

# Test your plugin specifically
./zpo plugins test-one your_plugin test_email.eml

# Test with all plugins
./zpo plugins test test_email.eml
```

### Performance Testing

Benchmark your plugin performance:

```go
func BenchmarkYourPlugin_AnalyzeContent(b *testing.B) {
    plugin := NewYourPlugin()
    config := &PluginConfig{
        Enabled: true,
        Settings: map[string]any{
            "api_key": "test-key",
        },
    }
    plugin.Initialize(config)
    
    testEmail := &email.Email{
        Subject: "Test Subject",
        Body:    "Test body content",
        From:    "test@example.com",
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := plugin.AnalyzeContent(ctx, testEmail)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Plugin Examples

### 1. Simple Keyword Plugin

```go
package plugins

type KeywordPlugin struct {
    config     *PluginConfig
    enabled    bool
    keywords   []string
    scores     map[string]float64
}

func NewKeywordPlugin() *KeywordPlugin {
    return &KeywordPlugin{
        keywords: []string{"urgent", "winner", "free", "money"},
        scores: map[string]float64{
            "urgent": 5.0,
            "winner": 8.0,
            "free":   3.0,
            "money":  4.0,
        },
    }
}

func (kp *KeywordPlugin) AnalyzeContent(ctx context.Context, email *email.Email) (*PluginResult, error) {
    result := &PluginResult{
        Name:       kp.Name(),
        Score:      0,
        Confidence: 0.7,
        Rules:      []string{},
        Metadata:   make(map[string]string),
    }
    
    content := strings.ToLower(email.Subject + " " + email.Body)
    
    for _, keyword := range kp.keywords {
        if strings.Contains(content, keyword) {
            score := kp.scores[keyword]
            result.Score += score
            result.Rules = append(result.Rules, fmt.Sprintf("Keyword '%s' found (+%.1f)", keyword, score))
        }
    }
    
    result.Metadata["keywords_checked"] = fmt.Sprintf("%d", len(kp.keywords))
    result.Metadata["keywords_found"] = fmt.Sprintf("%d", len(result.Rules))
    
    return result, nil
}
```

### 2. HTTP API Plugin

```go
package plugins

type HTTPAPIPlugin struct {
    config   *PluginConfig
    enabled  bool
    endpoint string
    apiKey   string
    client   *http.Client
}

func (hp *HTTPAPIPlugin) Analyze(ctx context.Context, email *email.Email) (*PluginResult, error) {
    result := &PluginResult{
        Name:       hp.Name(),
        Score:      0,
        Confidence: 0.8,
        Metadata:   make(map[string]string),
    }
    
    // Call external API
    response, err := hp.callAPI(ctx, email)
    if err != nil {
        result.Error = err
        return result, nil
    }
    
    result.Score = response.SpamScore
    result.Confidence = response.Confidence
    
    if response.IsSpam {
        result.Rules = append(result.Rules, "External API classified as spam")
    }
    
    result.Metadata["api_response_time"] = fmt.Sprintf("%.2fms", response.ResponseTime)
    result.Metadata["api_version"] = response.Version
    
    return result, nil
}
```

## Best Practices

### 1. Performance
- Keep plugin execution time under 100ms for most cases
- Use context timeouts appropriately
- Implement connection pooling for external APIs
- Cache frequently accessed data

### 2. Error Handling
- Never panic in plugin code
- Always return PluginResult even on errors
- Log errors for debugging
- Provide meaningful error messages

### 3. Configuration
- Use sensible defaults
- Validate configuration on initialization
- Support hot-reloading where possible
- Document all configuration options

### 4. Security
- Validate all inputs
- Use HTTPS for external APIs
- Handle API keys securely
- Implement rate limiting

### 5. Monitoring
- Track plugin performance metrics
- Log important events
- Provide health check endpoints
- Monitor external dependencies

## Deployment

### Building with Custom Plugins

```bash
# Build ZPO with your custom plugins
go build -o zpo .

# Test configuration
./zpo plugins list

# Test your plugin
./zpo plugins test-one your_plugin examples/test.eml
```

### Production Deployment

```yaml
# config-production.yaml
plugins:
  your_plugin:
    enabled: true
    weight: 2.0
    priority: 5
    timeout_ms: 2000
    settings:
      api_key: "${YOUR_PLUGIN_API_KEY}"
      endpoint: "https://api.yourservice.com/v1/analyze"
      max_retries: 3
      cache_ttl: 300
```

### Monitoring in Production

```bash
# Check plugin status
./zpo plugins stats

# Monitor plugin performance
tail -f logs/zpo.log | grep your_plugin

# Health checks
curl http://localhost:8080/health/plugins
```

## Troubleshooting

### Common Issues

1. **Plugin not found**: Ensure plugin is registered in both `spam_filter.go` and `cmd/plugins.go`
2. **Configuration errors**: Check YAML syntax and required fields
3. **Performance issues**: Use profiling tools and implement timeouts
4. **External API failures**: Implement retry logic and circuit breakers

### Debug Mode

Enable debug logging to see detailed plugin execution:

```yaml
logging:
  level: debug
  plugins:
    your_plugin: debug
```

This comprehensive guide should help you create powerful, efficient, and maintainable custom plugins for ZPO! 🚀 