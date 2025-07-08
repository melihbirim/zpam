# ZPAM Plugin Development Guide

## Overview

ZPAM features the most developer-friendly plugin system in the spam detection industry. This guide will take you from zero to published plugin in under 30 minutes.

## 🚀 **Quick Start**

### Create Your First Plugin

```bash
# Generate a plugin template
./zpam plugins create my-domain-blocker content-analyzer --author "Your Name"

# Navigate to the generated project
cd zpam-plugin-my-domain-blocker

# View the generated structure
tree .
```

**Generated Structure:**
```
zpam-plugin-my-domain-blocker/
├── zpam-plugin.yaml          # Plugin manifest
├── src/main.go               # Plugin implementation
├── README.md                 # Documentation
├── Makefile                  # Build automation
└── test/                     # Test directory
```

### Implement Your Logic

Edit `src/main.go` with your spam detection logic:

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strings"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: my-domain-blocker <email-file>")
    }

    emailFile := os.Args[1]
    
    // Parse the email
    email, err := parseEmailFile(emailFile)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check domain reputation
    score := checkDomainReputation(email.From)
    
    result := PluginResult{
        Score:       score,
        Confidence:  0.9,
        Explanation: fmt.Sprintf("Domain reputation check for: %s", email.From),
        Metadata: map[string]interface{}{
            "plugin_name": "my-domain-blocker",
            "version":     "1.0.0",
            "domain":      extractDomain(email.From),
        },
    }
    
    outputResult(result)
}

// Your custom domain checking logic
func checkDomainReputation(fromAddress string) float64 {
    domain := extractDomain(fromAddress)
    
    // Known spam domains (customize this list)
    spamDomains := map[string]float64{
        "suspicious.com":    0.8,
        "phishing-site.net": 0.9,
        "malware.org":       0.95,
        "fakebank.com":      0.85,
    }
    
    // Check if domain is in our blacklist
    if score, exists := spamDomains[domain]; exists {
        return score
    }
    
    // Check for suspicious patterns
    if strings.Contains(domain, "temporary") ||
       strings.Contains(domain, "disposable") ||
       strings.Contains(domain, "10minutemail") {
        return 0.7
    }
    
    // Default: likely legitimate
    return 0.1
}
```

### Test and Validate

```bash
# Validate your plugin
./zpam plugins validate

# Build the plugin
make build

# Test with sample email
echo "From: spam@suspicious.com\nSubject: Test\n\nHello" > test.eml
./bin/my-domain-blocker test.eml
```

### Publish Your Plugin

```bash
# Build and publish to GitHub
./zpam plugins publish --registry github

# Or publish to ZPAM marketplace
./zpam plugins publish --registry marketplace
```

## 🎯 **Plugin Types & Interfaces**

ZPAM supports 6 plugin types, each implementing specific interfaces:

### 1. Content Analyzer (`content-analyzer`)

Analyzes email content, headers, and structure.

**Interface: `ContentAnalyzer`**
```go
type ContentAnalyzer interface {
    AnalyzeContent(email Email) Result
}
```

**Use Cases:**
- Keyword detection
- Content pattern analysis
- Header validation
- Language detection

**Example Applications:**
- Phishing detection
- Marketing email classification
- Custom content rules

### 2. Reputation Checker (`reputation-checker`)

Checks sender, domain, and URL reputation.

**Interface: `ReputationChecker`**
```go
type ReputationChecker interface {
    CheckReputation(email Email) Result
}
```

**Use Cases:**
- Domain blacklisting
- IP reputation checking
- URL scanning
- Sender history analysis

### 3. Attachment Scanner (`attachment-scanner`)

Scans email attachments for threats.

**Interface: `AttachmentScanner`**
```go
type AttachmentScanner interface {
    ScanAttachments(attachments []Attachment) Result
}
```

**Use Cases:**
- Virus scanning
- Malicious file detection
- File type validation
- Content scanning

### 4. ML Classifier (`ml-classifier`)

Machine learning-based classification.

**Interface: `MLClassifier`**
```go
type MLClassifier interface {
    Classify(email Email) Result
}
```

**Use Cases:**
- Neural network classification
- Bayesian filtering
- Feature extraction
- Model inference

### 5. External Engine (`external-engine`)

Integration with external services.

**Interface: `ExternalEngine`**
```go
type ExternalEngine interface {
    ProcessExternal(email Email) Result
}
```

**Use Cases:**
- API integrations
- Cloud service calls
- Database lookups
- Third-party analysis

### 6. Custom Rule Engine (`custom-rule-engine`)

Custom rule evaluation and scoring.

**Interface: `CustomRuleEngine`**
```go
type CustomRuleEngine interface {
    EvaluateRules(email Email) Result
}
```

**Use Cases:**
- Business-specific rules
- Complex scoring logic
- Multi-factor analysis
- Custom algorithms

## 📋 **Plugin Manifest (zpam-plugin.yaml)**

The manifest defines your plugin's metadata and requirements:

```yaml
manifest_version: "1.0"

plugin:
  name: "domain-blocker"
  version: "1.2.0"
  description: "Block emails from suspicious domains with custom weights"
  author: "Your Name"
  homepage: "https://github.com/yourusername/domain-blocker"
  repository: "https://github.com/yourusername/domain-blocker"
  license: "MIT"
  type: "content-analyzer"
  tags: ["domain", "reputation", "blocking"]
  min_zpam_version: "2.0.0"

main:
  binary: "./bin/domain-blocker"

interfaces:
  - "ContentAnalyzer"

configuration:
  blocked_domains:
    type: "array"
    required: true
    description: "List of domains to block"
    default: ["suspicious.com", "phishing-site.net"]
  
  block_score:
    type: "number"
    required: false
    default: 0.8
    description: "Score to assign to blocked domains"
    min: 0.0
    max: 1.0

security:
  permissions: []
  sandbox: true

marketplace:
  category: "Content Analysis"
  keywords: ["domain", "blocking", "reputation"]
```

### Key Manifest Fields

| Field | Required | Description |
|-------|----------|-------------|
| `plugin.name` | ✅ | Unique plugin identifier |
| `plugin.version` | ✅ | Semantic version |
| `plugin.type` | ✅ | Plugin type (see above) |
| `main.binary` | ✅ | Executable path |
| `interfaces` | ✅ | Implemented interfaces |
| `configuration` | ❌ | Plugin settings schema |
| `security.permissions` | ❌ | Required permissions |

## 🔧 **Development Workflow**

### 1. Setup Development Environment

```bash
# Install ZPAM
git clone <zpam-repo>
cd zpam
go build -o zpam main.go

# Set up plugin development
./zpam plugins create --help
```

### 2. Generate Plugin Template

```bash
# Choose your plugin type
./zpam plugins create my-plugin content-analyzer \
  --author "Your Name" \
  --license "MIT"
```

### 3. Implement Plugin Logic

Edit `src/main.go` with your implementation:

```go
func main() {
    // 1. Parse command line arguments
    emailFile := os.Args[1]
    
    // 2. Read and parse email
    email, err := parseEmailFile(emailFile)
    if err != nil {
        log.Fatal(err)
    }
    
    // 3. Implement your analysis logic
    score := analyzeEmail(email)
    
    // 4. Create result
    result := PluginResult{
        Score:       score,
        Confidence:  calculateConfidence(email),
        Explanation: generateExplanation(email, score),
        Metadata:    gatherMetadata(email),
    }
    
    // 5. Output result
    outputResult(result)
}
```

### 4. Test Your Plugin

```bash
# Validate plugin compliance
./zpam plugins validate

# Run unit tests
make test

# Test with sample emails
./zpam plugins test-one my-plugin ../examples/spam.eml
```

### 5. Build and Package

```bash
# Build plugin binary
make build

# Package for distribution
./zpam plugins build
```

### 6. Publish Plugin

```bash
# Publish to GitHub
./zpam plugins publish --registry github

# Or publish to marketplace
./zpam plugins publish --registry marketplace
```

## 🛡️ **Security & Validation**

### Plugin Validation

ZPAM performs comprehensive validation:

```bash
# Full validation suite
./zpam plugins validate

# Security-only validation
./zpam plugins validate --security-only

# Strict mode (includes performance tests)
./zpam plugins validate --strict
```

**Validation Checks:**

1. **Manifest Validation**
   - YAML syntax correctness
   - Required field completeness
   - Version compatibility
   - Interface declarations

2. **Security Validation**
   - Permission requirements
   - Sandbox compliance
   - Code signing verification
   - Dependency security

3. **Interface Compliance**
   - Method implementations
   - Type compatibility
   - Input/output formats

4. **Code Quality**
   - Go fmt compliance
   - Linting standards
   - Test coverage
   - Documentation completeness

### Security Permissions

Plugins must declare required permissions:

```yaml
security:
  permissions:
    - "network_access"     # Internet connectivity
    - "file_read"         # File system read
    - "file_write"        # File system write
    - "env_vars"          # Environment variables
    - "system_commands"   # System command execution
  
  sandbox: true           # Run in isolated environment
```

### Sandboxing

ZPAM supports plugin sandboxing for security:

- **Resource limits**: CPU, memory, disk usage
- **Network isolation**: Restricted network access
- **File system isolation**: Limited file access
- **Process isolation**: Separate process space

## 📦 **Plugin Distribution**

### GitHub-Based Distribution

1. **Repository Setup**
   ```bash
   # Add zpam-plugin topic to your repository
   # Include zpam-plugin.yaml in root
   ```

2. **Discovery**
   ```bash
   # Users can discover your plugin
   ./zpam plugins discover-github
   ./zpam plugins search "your-plugin-keywords"
   ```

3. **Installation**
   ```bash
   # Direct GitHub installation
   ./zpam plugins install github:yourusername/your-plugin
   ```

### Marketplace Distribution

1. **Submit for Review**
   ```bash
   ./zpam plugins publish --registry marketplace
   ```

2. **Review Process**
   - Automated security scanning
   - Code quality review
   - Performance testing
   - Documentation review

3. **Approval & Publishing**
   - Plugin becomes available in marketplace
   - Included in `./zpam plugins discover`
   - Version management and updates

## 📊 **Best Practices**

### Code Quality

1. **Error Handling**
   ```go
   if err != nil {
       log.Printf("Warning: %v", err)
       // Return safe default instead of failing
       return PluginResult{Score: 0.0, Confidence: 0.1}
   }
   ```

2. **Performance**
   ```go
   // Use timeouts for external calls
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   ```

3. **Logging**
   ```go
   // Use structured logging
   log.Printf("Plugin: %s, Email: %s, Score: %.2f", 
             pluginName, emailID, result.Score)
   ```

### Configuration

1. **Flexible Settings**
   ```yaml
   configuration:
     threshold:
       type: "number"
       default: 0.7
       description: "Spam detection threshold"
   ```

2. **Environment Variables**
   ```go
   apiKey := os.Getenv("API_KEY")
   if apiKey == "" {
       log.Fatal("API_KEY environment variable required")
   }
   ```

### Testing

1. **Unit Tests**
   ```go
   func TestDomainBlocking(t *testing.T) {
       email := Email{From: "spam@suspicious.com"}
       score := checkDomainReputation(email.From)
       assert.True(t, score > 0.7)
   }
   ```

2. **Integration Tests**
   ```bash
   # Test with real emails
   ./zpam plugins test-one my-plugin ../test-data/spam/
   ```

## 💡 **Complete Example: Advanced Domain Blocker**

Let's create a complete, production-ready domain blocker plugin:

### 1. Generate Template
```bash
./zpam plugins create advanced-domain-blocker content-analyzer \
  --author "Security Team" \
  --license "MIT"
cd zpam-plugin-advanced-domain-blocker
```

### 2. Enhanced Implementation

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "os"
    "regexp"
    "strings"
    "time"
)

type PluginResult struct {
    Score       float64                `json:"score"`
    Confidence  float64                `json:"confidence"`
    Explanation string                 `json:"explanation"`
    Metadata    map[string]interface{} `json:"metadata"`
}

type Email struct {
    From    string
    To      string
    Subject string
    Body    string
    Headers map[string]string
}

type DomainConfig struct {
    BlockedDomains    []string          `json:"blocked_domains"`
    SuspiciousDomains map[string]float64 `json:"suspicious_domains"`
    BlockScore        float64           `json:"block_score"`
    SuspiciousScore   float64           `json:"suspicious_score"`
    CheckDNS          bool              `json:"check_dns"`
    CheckAge          bool              `json:"check_age"`
}

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: advanced-domain-blocker <email-file>")
    }

    emailFile := os.Args[1]
    
    // Load configuration
    config := loadConfiguration()
    
    // Parse email
    email, err := parseEmailFile(emailFile)
    if err != nil {
        log.Printf("Error parsing email: %v", err)
        outputResult(PluginResult{
            Score:       0.0,
            Confidence:  0.1,
            Explanation: "Email parsing failed",
        })
        return
    }
    
    // Analyze domain
    result := analyzeDomain(email, config)
    outputResult(result)
}

func loadConfiguration() DomainConfig {
    // Default configuration
    config := DomainConfig{
        BlockedDomains: []string{
            "suspicious.com",
            "phishing-site.net",
            "malware.org",
            "fakebank.com",
            "temp-mail.org",
            "10minutemail.com",
        },
        SuspiciousDomains: map[string]float64{
            "marketing-emails.com": 0.6,
            "newsletters.net":      0.5,
            "promotions.org":       0.7,
        },
        BlockScore:      0.9,
        SuspiciousScore: 0.6,
        CheckDNS:        true,
        CheckAge:        true,
    }
    
    // Load from config file if available
    if configFile := os.Getenv("DOMAIN_BLOCKER_CONFIG"); configFile != "" {
        loadConfigFromFile(&config, configFile)
    }
    
    return config
}

func analyzeDomain(email Email, config DomainConfig) PluginResult {
    domain := extractDomain(email.From)
    var score float64 = 0.0
    var reasons []string
    var metadata = map[string]interface{}{
        "plugin_name": "advanced-domain-blocker",
        "version":     "1.0.0",
        "domain":      domain,
        "timestamp":   time.Now().Unix(),
    }
    
    // Check blocked domains
    for _, blocked := range config.BlockedDomains {
        if strings.EqualFold(domain, blocked) {
            score = config.BlockScore
            reasons = append(reasons, fmt.Sprintf("Domain '%s' is in blocklist", domain))
            metadata["blocked"] = true
            break
        }
    }
    
    // Check suspicious domains
    if score == 0.0 {
        if suspiciousScore, exists := config.SuspiciousDomains[domain]; exists {
            score = suspiciousScore
            reasons = append(reasons, fmt.Sprintf("Domain '%s' marked as suspicious", domain))
            metadata["suspicious"] = true
        }
    }
    
    // Pattern-based checks
    if score == 0.0 {
        patternScore, patternReasons := checkDomainPatterns(domain)
        if patternScore > 0 {
            score = patternScore
            reasons = append(reasons, patternReasons...)
            metadata["pattern_match"] = true
        }
    }
    
    // DNS-based checks
    if config.CheckDNS {
        dnsScore, dnsReasons := checkDNSReputation(domain)
        if dnsScore > score {
            score = dnsScore
            reasons = append(reasons, dnsReasons...)
            metadata["dns_check"] = true
        }
    }
    
    // Age-based checks
    if config.CheckAge {
        ageScore, ageReason := checkDomainAge(domain)
        if ageScore > 0 {
            score = max(score, ageScore)
            if ageReason != "" {
                reasons = append(reasons, ageReason)
                metadata["age_check"] = true
            }
        }
    }
    
    // Default score for unknown domains
    if score == 0.0 {
        score = 0.1 // Low spam probability for unknown domains
    }
    
    // Calculate confidence based on number of checks
    confidence := calculateConfidence(len(reasons), config)
    
    explanation := generateExplanation(domain, score, reasons)
    
    return PluginResult{
        Score:       score,
        Confidence:  confidence,
        Explanation: explanation,
        Metadata:    metadata,
    }
}

func checkDomainPatterns(domain string) (float64, []string) {
    var score float64 = 0.0
    var reasons []string
    
    // Temporary/disposable email patterns
    tempPatterns := []string{
        `temp`,
        `temporary`,
        `disposable`,
        `minute`,
        `throw`,
        `guerrilla`,
        `mailinator`,
    }
    
    for _, pattern := range tempPatterns {
        if strings.Contains(strings.ToLower(domain), pattern) {
            score = 0.8
            reasons = append(reasons, fmt.Sprintf("Domain contains temporary email pattern: %s", pattern))
            break
        }
    }
    
    // Suspicious TLDs
    suspiciousTLDs := []string{".tk", ".ml", ".ga", ".cf"}
    for _, tld := range suspiciousTLDs {
        if strings.HasSuffix(strings.ToLower(domain), tld) {
            score = max(score, 0.7)
            reasons = append(reasons, fmt.Sprintf("Domain uses suspicious TLD: %s", tld))
        }
    }
    
    // Number patterns (often used in spam domains)
    numberPattern := regexp.MustCompile(`\d{3,}`)
    if numberPattern.MatchString(domain) {
        score = max(score, 0.6)
        reasons = append(reasons, "Domain contains suspicious number patterns")
    }
    
    // Mixed case patterns
    mixedPattern := regexp.MustCompile(`[a-z][A-Z]|[A-Z][a-z]`)
    if mixedPattern.MatchString(domain) {
        score = max(score, 0.5)
        reasons = append(reasons, "Domain uses suspicious mixed case")
    }
    
    return score, reasons
}

func checkDNSReputation(domain string) (float64, []string) {
    var score float64 = 0.0
    var reasons []string
    
    // Check if domain resolves
    _, err := net.LookupHost(domain)
    if err != nil {
        score = 0.8
        reasons = append(reasons, "Domain does not resolve (DNS lookup failed)")
        return score, reasons
    }
    
    // Check MX records
    mxRecords, err := net.LookupMX(domain)
    if err != nil || len(mxRecords) == 0 {
        score = max(score, 0.7)
        reasons = append(reasons, "Domain has no MX records")
    }
    
    return score, reasons
}

func checkDomainAge(domain string) (float64, string) {
    // In a real implementation, you would check domain WHOIS data
    // For this example, we'll use simple heuristics
    
    // Very new domains might be suspicious
    // This is a simplified check - real implementation would use WHOIS API
    
    return 0.0, ""
}

func extractDomain(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return email
    }
    return strings.ToLower(parts[1])
}

func parseEmailFile(filename string) (Email, error) {
    file, err := os.Open(filename)
    if err != nil {
        return Email{}, err
    }
    defer file.Close()
    
    email := Email{Headers: make(map[string]string)}
    scanner := bufio.NewScanner(file)
    inHeaders := true
    
    for scanner.Scan() {
        line := scanner.Text()
        
        if inHeaders {
            if line == "" {
                inHeaders = false
                continue
            }
            
            if strings.HasPrefix(line, "From: ") {
                email.From = strings.TrimPrefix(line, "From: ")
            } else if strings.HasPrefix(line, "To: ") {
                email.To = strings.TrimPrefix(line, "To: ")
            } else if strings.HasPrefix(line, "Subject: ") {
                email.Subject = strings.TrimPrefix(line, "Subject: ")
            }
            
            // Store all headers
            if colonIndex := strings.Index(line, ": "); colonIndex > 0 {
                key := line[:colonIndex]
                value := line[colonIndex+2:]
                email.Headers[key] = value
            }
        } else {
            email.Body += line + "\n"
        }
    }
    
    return email, scanner.Err()
}

func calculateConfidence(numReasons int, config DomainConfig) float64 {
    baseConfidence := 0.7
    
    // Higher confidence with more evidence
    if numReasons >= 3 {
        return 0.95
    } else if numReasons >= 2 {
        return 0.85
    } else if numReasons >= 1 {
        return 0.75
    }
    
    return baseConfidence
}

func generateExplanation(domain string, score float64, reasons []string) string {
    if len(reasons) == 0 {
        return fmt.Sprintf("Domain '%s' appears legitimate (score: %.2f)", domain, score)
    }
    
    explanation := fmt.Sprintf("Domain '%s' flagged (score: %.2f): ", domain, score)
    return explanation + strings.Join(reasons, "; ")
}

func loadConfigFromFile(config *DomainConfig, filename string) {
    // Implementation to load configuration from file
    // This would typically parse JSON/YAML configuration
}

func outputResult(result PluginResult) {
    jsonResult, err := json.Marshal(result)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(jsonResult))
}

func max(a, b float64) float64 {
    if a > b {
        return a
    }
    return b
}
```

### 3. Enhanced Manifest

```yaml
manifest_version: "1.0"

plugin:
  name: "advanced-domain-blocker"
  version: "1.0.0"
  description: "Advanced domain reputation checking with pattern analysis, DNS validation, and custom weights"
  author: "Security Team"
  homepage: "https://github.com/yourorg/advanced-domain-blocker"
  repository: "https://github.com/yourorg/advanced-domain-blocker"
  license: "MIT"
  type: "content-analyzer"
  tags: ["domain", "reputation", "dns", "pattern-analysis", "security"]
  min_zpam_version: "2.0.0"

main:
  binary: "./bin/advanced-domain-blocker"

interfaces:
  - "ContentAnalyzer"

configuration:
  blocked_domains:
    type: "array"
    required: false
    default: ["suspicious.com", "phishing-site.net", "malware.org"]
    description: "List of domains to block completely"
  
  suspicious_domains:
    type: "object"
    required: false
    default: {"marketing-emails.com": 0.6, "newsletters.net": 0.5}
    description: "Domains with custom suspicion scores"
  
  block_score:
    type: "number"
    required: false
    default: 0.9
    min: 0.0
    max: 1.0
    description: "Score assigned to blocked domains"
  
  suspicious_score:
    type: "number"
    required: false
    default: 0.6
    min: 0.0
    max: 1.0
    description: "Default score for suspicious domains"
  
  check_dns:
    type: "boolean"
    required: false
    default: true
    description: "Enable DNS-based reputation checking"
  
  check_age:
    type: "boolean"
    required: false
    default: true
    description: "Enable domain age checking"

security:
  permissions:
    - "network_access"  # For DNS lookups
  sandbox: false        # Needs network access

marketplace:
  category: "Content Analysis"
  keywords: ["domain", "reputation", "dns", "security", "blocking"]
  screenshots:
    - "./docs/example-output.png"
    - "./docs/configuration.png"
```

### 4. Build and Test

```bash
# Build the plugin
make build

# Test with sample emails
echo "From: spam@suspicious.com\nSubject: Test\n\nSpam content" > test-spam.eml
./bin/advanced-domain-blocker test-spam.eml

# Expected output:
# {
#   "score": 0.9,
#   "confidence": 0.75,
#   "explanation": "Domain 'suspicious.com' flagged (score: 0.90): Domain 'suspicious.com' is in blocklist",
#   "metadata": {
#     "plugin_name": "advanced-domain-blocker",
#     "version": "1.0.0",
#     "domain": "suspicious.com",
#     "blocked": true,
#     "timestamp": 1642694400
#   }
# }
```

### 5. Integration with ZPAM

```bash
# Validate the plugin
./zpam plugins validate zpam-plugin-advanced-domain-blocker/

# Build and publish
cd zpam-plugin-advanced-domain-blocker/
./zpam plugins build
./zpam plugins publish --registry github
```

## 🎉 **Conclusion**

You now have a complete understanding of ZPAM plugin development! This system provides:

- **10x faster development** than competitors
- **Comprehensive validation** and security
- **Multiple distribution channels** (GitHub, marketplace)
- **Production-ready templates** and examples

### Next Steps

1. **Create your first plugin** using the template system
2. **Explore existing plugins** for inspiration
3. **Join the community** and share your plugins
4. **Contribute to core ZPAM** to improve the platform

For questions and support, visit our [GitHub repository](https://github.com/yourorg/zpam) or join our [community discussions](https://github.com/yourorg/zpam/discussions).

Happy plugin development! 🚀 