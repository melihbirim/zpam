# ZPAM - Lightning-Fast Spam Filter 🫏

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Speed](https://img.shields.io/badge/Speed-%3C5ms_per_email-brightgreen.svg)](#performance)
[![Setup](https://img.shields.io/badge/Setup-%3C5_minutes-orange.svg)](#quick-start)

ZPAM is a **zero-configuration spam filter** that gets you detecting spam in under 5 minutes. Named after a baby donkey - it's **free, fast, and reliable**.

> **🎯 Mission**: Make spam detection as simple as `npm install` but as powerful as enterprise security suites.

## 🚀 **Get Started in Under 5 Minutes**

### **Option 1: Zero-Config Install (Recommended)**
```bash
# One command sets up everything automatically
./zpam install

# Test immediately with provided samples
./zpam test training-data/spam/06_spam_phishing.eml

# Start real-time monitoring
./zpam monitor
```

### **Option 2: Interactive Setup**
```bash
# Interactive wizard with guided configuration
./zpam quickstart

# Follow the prompts to customize your setup
```

**That's it!** ZPAM automatically:
- 🔍 Detects your system capabilities (Redis, Docker, etc.)
- ⚙️ Generates optimal configuration
- 🧠 Sets up learning backend (Redis or file-based)
- 📧 Creates sample emails for testing
- ✅ Validates everything works

## ✨ **Key Features**

### **🎛️ Zero-Config Setup**
- **Auto-Detection**: Automatically discovers Redis, Docker, SpamAssassin, Postfix
- **Smart Configuration**: Generates optimal settings based on your system
- **Instant Success**: From download to detecting spam in under 5 minutes
- **Guided Setup**: Interactive wizard for customization

### **⚡ Lightning Performance** 
- **Sub-5ms Processing**: Ultra-fast email analysis
- **Redis-Backed Learning**: High-performance Bayesian classification
- **Real-time Monitoring**: Live dashboards and alerts
- **Service Management**: Start/stop/restart with built-in health checks

### **🧠 Advanced Learning**
- **Enhanced Training**: Auto-discovery, progress tracking, resume capability
- **Multi-Backend Support**: Redis or file-based learning
- **Accuracy Analytics**: Before/after training comparisons
- **Session Management**: Pause and resume training sessions

### **🔧 Production Ready**
- **Milter Integration**: Real-time filtering for Postfix/Sendmail
- **Service Management**: Full lifecycle management (start/stop/restart/reload)
- **Health Monitoring**: Comprehensive status dashboards
- **Plugin System**: Extensible with SpamAssassin, Rspamd, custom rules

## 📊 **System Status at a Glance**

```bash
# Real-time system dashboard
./zpam status

# Live monitoring with charts
./zpam monitor

# Service management
./zpam start
./zpam stop
./zpam restart
```

**Example Status Output:**
```
🫏 ZPAM System Status Dashboard
═══════════════════════════════════════

🚀 Service Status
  Status: ✅ Running (PID: 1234)
  Uptime: 2h 15m
  Config: config-quickstart.yaml

📊 Performance  
  Emails processed: 1,247
  Average time: 1.2ms/email
  Throughput: 45 emails/min

🧠 Learning Status
  Backend: Redis (connected)
  Spam learned: 156 emails
  Ham learned: 203 emails
  
🏥 Health: ✅ HEALTHY
```

## 🎯 **Email Classification**

ZPAM uses an intuitive 1-5 scoring system:

| Score | Classification | Action | Description |
|-------|---------------|--------|-------------|
| **1-2** | 🟢 **Clean** | Keep in inbox | Legitimate emails |
| **3** | 🟡 **Questionable** | Keep (review) | Newsletters, marketing |
| **4-5** | 🔴 **Spam** | Move to spam | Obvious spam and phishing |

### **Real Example Results**
```bash
# Business email
./zpam test training-data/ham/01_clean_business.eml
# → Score: 1/5 ✅ HAM

# Phishing attempt  
./zpam test training-data/spam/06_spam_phishing.eml
# → Score: 5/5 🚫 SPAM
```

## 🧠 **Enhanced Training System**

Train ZPAM with your email data using our advanced training system:

```bash
# Auto-discover and train from directory structure
./zpam train --auto-discover /path/to/emails

# Traditional spam/ham training
./zpam train --spam-dir spam/ --ham-dir clean/

# Interactive training with data preview
./zpam train --spam-dir spam/ --ham-dir clean/ --interactive

# Benchmark accuracy improvements
./zpam train --spam-dir spam/ --ham-dir clean/ --benchmark

# Resume interrupted training
./zpam train --resume
```

**Advanced Features:**
- 📊 **Live Progress Tracking**: Real-time progress bars and statistics
- 🔍 **Data Validation**: Check training data quality before processing
- 📈 **Accuracy Analytics**: Before/after training comparisons
- 💾 **Session Management**: Resume interrupted training sessions
- 🎯 **Optimization**: Auto-optimize training data selection

## 🔧 **Configuration Made Simple**

ZPAM automatically generates optimal configuration, but you can customize:

```yaml
# Auto-generated config-quickstart.yaml
learning:
  backend: "redis"              # Auto-detected: Redis or file
  
detection:
  spam_threshold: 4             # Customizable sensitivity
  
performance:
  max_concurrent_emails: 4      # Optimized for your system
  timeout_ms: 5000             # Balanced for accuracy vs speed

milter:
  enabled: true                # Auto-detected if Postfix available
  address: "127.0.0.1:7357"   # Ready for integration
```

**Configuration Profiles:**
- `config-quickstart.yaml` - Auto-generated optimal settings
- `config.yaml` - Production configuration  
- `config-fast.yaml` - Speed-optimized
- `config-cached.yaml` - Memory-optimized

## 🎛️ **Service Management**

ZPAM includes full service lifecycle management:

```bash
# Start ZPAM service
./zpam start --mode milter  # For Postfix integration
./zpam start --mode standalone  # For testing

# Check service status
./zpam status

# Restart with new configuration
./zpam restart

# Reload configuration without restart
./zpam reload

# Stop service gracefully
./zpam stop
```

## 📊 **Real-Time Monitoring**

Monitor ZPAM performance with live dashboards:

```bash
# Live monitoring dashboard
./zpam monitor

# Compact view
./zpam monitor --compact

# Include live logs
./zpam monitor --logs

# Custom refresh interval
./zpam monitor --interval 1s
```

**Monitoring Features:**
- 📈 **Live Charts**: Email throughput and response times
- 🚨 **Smart Alerts**: Automatic issue detection and recommendations
- 📊 **Performance Metrics**: Memory, CPU, and network usage
- 🧠 **Learning Analytics**: Training progress and model accuracy

## 🔌 **Plugin System & Marketplace**

ZPAM features the industry's most comprehensive plugin ecosystem for spam detection. Build, discover, and deploy plugins in minutes using **Go or Lua** - making it accessible to everyone from beginners to enterprise developers.

### **🚀 Plugin Development - From Idea to Production**

Create production-ready plugins in under 10 lines of code using **Go** or **Lua**:

#### **🟦 Go Plugins (Enterprise Performance)**
```bash
# 1. Generate Go plugin template
./zpam plugins create my-domain-blocker content-analyzer go --author "Your Name"

# 2. Implement your logic (edit src/main.go)
cd zpam-plugin-my-domain-blocker

# 3. Validate & test
./zpam plugins validate
./zpam plugins test ../examples/spam.eml

# 4. Build & publish
./zpam plugins build
./zpam plugins publish --registry github
```

#### **🌙 Lua Plugins (Rapid Development)**
```bash
# 1. Generate Lua plugin template
./zpam plugins create my-keyword-filter content-analyzer lua

# 2. Implement your logic (edit src/main.lua)
cd zpam-plugin-my-keyword-filter

# 3. Validate & test  
./zpam plugins validate
./zpam plugins test ../examples/spam.eml

# 4. Deploy instantly (no compilation needed!)
./zpam plugins install .
```

**🎯 6 Plugin Types Available:**
- `content-analyzer` - Analyze email content and headers
- `reputation-checker` - Check sender/domain reputation  
- `attachment-scanner` - Scan email attachments
- `ml-classifier` - Machine learning classification
- `external-engine` - Integration with external services
- `custom-rule-engine` - Custom rule evaluation

### **📦 Plugin Discovery & Installation**

Install plugins from multiple sources:

```bash
# From marketplace
./zpam plugins discover
./zpam plugins install openai-classifier

# From GitHub
./zpam plugins install github:security-corp/phishing-detector
./zpam plugins install https://github.com/user/spam-plugin

# From local files
./zpam plugins install ./my-plugin/
./zpam plugins install plugin.zip

# Search for specific functionality
./zpam plugins search "phishing"
./zpam plugins search "ai"
```

### **🔍 Plugin Management**

```bash
# List installed plugins
./zpam plugins list

# Test specific plugin
./zpam plugins test-one spamassassin email.eml

# Enable/disable plugins
./zpam plugins enable spamassassin
./zpam plugins disable rspamd

# View plugin statistics
./zpam plugins stats

# Update plugin registry
./zpam plugins update-registry
```

### **🛡️ Security & Validation**

ZPAM includes comprehensive plugin validation:

```bash
# Validate plugin security and compliance
./zpam plugins validate ./my-plugin/

# Security-only scan
./zpam plugins validate --security-only

# Strict validation mode
./zpam plugins validate --strict
```

**Validation Includes:**
- ✅ **Manifest Compliance** - YAML syntax and required fields
- ✅ **Interface Verification** - Plugin implements declared interfaces
- ✅ **Security Scanning** - Permission analysis and sandbox compliance
- ✅ **Code Quality** - Linting, testing, documentation standards
- ✅ **Dependency Checking** - All dependencies available

### **📊 Available Plugin Ecosystem**

**Official Plugins:**
- **SpamAssassin**: Industry-standard spam detection
- **Rspamd**: Modern spam filtering engine  
- **OpenAI Classifier**: AI-powered spam detection
- **VirusTotal**: URL and attachment scanning
- **Microsoft Defender**: Enterprise integration

**Community Plugins:**
- **Phishing Detector Pro**: Advanced phishing detection
- **Custom Rules Engine**: User-defined detection rules
- **Slack Alerts**: Real-time notifications
- **Advanced Bayes**: Enhanced Bayesian filtering

### **🏗️ Multi-Language Plugin Architecture**

ZPAM supports plugins in multiple languages with unified interfaces:

#### **🟦 Go Plugins (Enterprise Performance)**
```go
// ContentAnalyzer interface
type ContentAnalyzer interface {
    AnalyzeContent(email Email) Result
}

// Example plugin result
type Result struct {
    Score       float64  // 0.0-100.0 (spam score)
    Confidence  float64  // 0.0-1.0 (confidence level)
    Rules       []string // Triggered rule descriptions
    Metadata    map[string]string // Additional data
}
```

#### **🌙 Lua Plugins (Rapid Development)**
```lua
-- Main function that ZPAM calls
function analyze_content(email)
    local result = {
        score = 0.0,       -- 0.0 to 100.0 (higher = more spam)
        confidence = 0.7,  -- 0.0 to 1.0 (confidence in score)
        rules = {},        -- Array of triggered rule descriptions
        metadata = {}      -- Key-value pairs of additional information
    }
    
    -- Your custom logic here
    if zpam.contains(email.subject, "URGENT") then
        result.score = 75.0
        table.insert(result.rules, "Urgent keyword detected")
    end
    
    return result
end
```

**Key Features:**
- 🌍 **Multi-Language Support** - Go for performance, Lua for simplicity
- 🔧 **Template Generation** - Complete project structure in seconds
- 🔒 **Security Sandboxing** - Isolated execution environment
- 📝 **Auto Documentation** - Generated README and examples
- 🎯 **Interface Compliance** - Type-safe plugin development
- 🚀 **One-Command Publishing** - Deploy to GitHub or marketplace
- ⚡ **Hot Reload** - Lua plugins update without restarts

### **💡 Example: Real Working Plugins**

#### **Go Domain Blocker Plugin**
```go
// Generated template in src/main.go
func main() {
    email := parseEmail(os.Args[1])
    
    // Your custom logic
    score := checkDomainReputation(email.From)
    
    result := PluginResult{
        Score:       score,
        Confidence:  0.9,
        Rules:       []string{fmt.Sprintf("Domain reputation: %s", email.From)},
        Metadata:    map[string]string{"plugin": "domain-blocker"},
    }
    
    outputJSON(result)
}
```

#### **Lua Spam Keywords Plugin**
```lua
-- Real working example from our test suite
function analyze_content(email)
    local spam_keywords = {"lottery", "congratulations", "urgent", "winner"}
    local result = {
        score = 0.0,
        confidence = 0.8,
        rules = {},
        metadata = {
            plugin_name = "lua-spam-keywords",
            version = "1.0.0"
        }
    }
    
    -- Check subject and body for spam keywords
    for _, keyword in ipairs(spam_keywords) do
        if zpam.contains(email.subject, keyword) then
            result.score = result.score + 30.0
            table.insert(result.rules, "Spam keyword in subject")
        end
        if zpam.contains(email.body, keyword) then
            result.score = result.score + 25.0  
            table.insert(result.rules, "Spam keyword in body")
        end
    end
    
    return result
end
```

**🎯 Real Performance Results:**
```bash
# Testing the Lua spam keywords plugin
./zpam plugins test-one lua-spam-keywords test-spam.eml

✓ lua-spam-keywords    Score: 130.00  Confidence: 0.80  Time: 321µs
  Rules: [Spam keyword in subject, Spam keyword in body]
```

#### **🛠️ Lua Plugin API**

Lua plugins have access to powerful ZPAM API functions:

```lua
-- String utilities
zpam.contains(text, pattern)        -- Case-insensitive substring search
zpam.regex_match(text, pattern)     -- Regular expression matching

-- Email utilities  
zpam.domain_from_email(email_addr)  -- Extract domain from email address

-- Logging
zpam.log(message)                   -- Log messages for debugging
```

**Email Data Structure Available to Lua:**
```lua
-- The email parameter contains:
email.from        -- Sender email address
email.to          -- Recipient email address (comma-separated)
email.subject     -- Email subject line
email.body        -- Plain text email body
email.headers     -- Email headers (table)
email.attachments -- Email attachments (table)
```

See **[Plugin Development Guide](docs/plugin-development.md)** for complete examples and tutorials.

## 📮 **Milter Integration**

Real-time email filtering for mail servers:

```bash
# ZPAM auto-configures milter if Postfix is detected
./zpam milter --config config-quickstart.yaml
```

**Postfix Configuration:**
```conf
# /etc/postfix/main.cf (auto-detected during install)
smtpd_milters = inet:localhost:7357
non_smtpd_milters = inet:localhost:7357
```

## 🧪 **Testing & Validation**

ZPAM includes comprehensive testing tools:

```bash
# Test single email
./zpam test email.eml

# Validate headers (SPF/DKIM/DMARC)
./zpam headers email.eml

# Run benchmark tests
./zpam benchmark --input test-emails/

# DNS testing tools
./zpam dnstest demo
```

## 🐳 **Docker Deployment**

For containerized environments:

```bash
# Quick Docker deployment
docker run -d \
  -p 7357:7357 \
  -v ./config-quickstart.yaml:/app/config.yaml \
  zpam:latest

# Or use our auto-generated Docker setup
./zpam install --docker
```

## 📚 **Documentation**

- 🚀 **[Quick Start Guide](docs/QUICKSTART.md)** - Get running in 5 minutes
- 🔧 **[Configuration Reference](docs/CONFIG.md)** - Complete settings guide
- 🧠 **[Training Guide](training-data/README.md)** - Optimize accuracy
- 🔌 **[Plugin Development](docs/plugin-development.md)** - Build and publish custom plugins
- 📊 **[Performance Tuning](docs/PERFORMANCE.md)** - Optimize for your needs

## 🗺️ **Roadmap**

- ✅ **Phase 1.1**: Zero-Config Quick Start *(Completed)*
- 🔄 **Phase 1.2**: Plugin Marketplace *(In Progress)*
- 📅 **Phase 1.3**: Visual Configuration Interface
- 📅 **Phase 2**: Enhanced Developer Experience

*See [ROADMAP.md](docs/ROADMAP.md) for detailed development plans.*

## 🤝 **Contributing**

We welcome contributions! ZPAM is designed to be:
- **Simple**: Easy to understand and modify
- **Fast**: Performance-first architecture
- **Reliable**: Production-ready from day one

```bash
# Get involved
git clone <repository-url>
cd zpam
./zpam install  # Get started in minutes
```

## 📄 **License**

MIT License - completely free for personal and commercial use.

---

**🫏 ZPAM: Because spam filtering should be as reliable as a donkey, and faster than you'd expect.**

*From zero to spam-free in under 5 minutes.* 