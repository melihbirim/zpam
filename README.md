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

## 🔌 **Plugin System**

Extend ZPAM with powerful plugins:

```bash
# List available plugins
./zpam plugins list

# Test specific plugin
./zpam plugins test-one spamassassin email.eml

# Enable/disable plugins
./zpam plugins enable spamassassin
./zpam plugins disable rspamd

# View plugin statistics
./zpam plugins stats
```

**Available Plugins:**
- **SpamAssassin**: Industry-standard spam detection
- **Rspamd**: Modern spam filtering engine
- **Custom Rules**: User-defined detection rules
- **VirusTotal**: Reputation checking
- **Machine Learning**: Advanced ML classification

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
- 🔌 **[Plugin Development](docs/custom_plugins.md)** - Build custom plugins
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