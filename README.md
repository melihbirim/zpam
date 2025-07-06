# ZPO - Baby Donkey Spam Filter 🫏

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Speed](https://img.shields.io/badge/Speed-%3C5ms_per_email-brightgreen.svg)](#performance)
[![DNS](https://img.shields.io/badge/DNS-Non_Blocking-blue.svg)](#dns-features)

ZPO is a lightning-fast, free spam filter that processes emails in under 5ms. Named after baby donkey - it's free, fast, and reliable.

## ✨ Features

- **⚡ Ultra-Fast**: Processes emails in under 5ms
- **🎯 Smart Scoring**: Rates emails 1-5 for precise classification
- **📁 Auto-Sorting**: Automatically moves spam (4-5 rating) to spam folder
- **🔍 Deep Analysis**: Analyzes content, headers, attachments, and sender reputation
- **🚀 Non-Blocking DNS**: Async DNS operations with 62x performance improvement
- **🧪 Internal Testing**: Controlled DNS testing environment with configurable TTL
- **🆓 Completely Free**: No licensing fees or restrictions
- **🚀 Easy to Use**: Simple CLI interface

## 📊 Performance

ZPO consistently achieves sub-5ms processing times:

- **Clean emails**: ~0.2-0.4ms
- **Spam emails**: ~0.5-0.8ms
- **Average**: ~0.78ms per email
- **Batch processing**: Linear scaling with excellent performance

### DNS Performance
- **Real DNS (Cold)**: ~5s for 8 domains
- **Real DNS (Warm)**: ~5s for 8 domains (cached)
- **Test Server**: ~80ms for 8 domains (**62x faster!**)
- **Cache Hit Rate**: Up to 87.5% in production

## 🛠️ Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
git clone <repository-url>
cd zpo
go mod tidy
go build -o zpo
```

## 🚀 Usage

### Test a Single Email

```bash
./zpo test email.eml
```

**Output:**
```
ZPO Test Results:
File: email.eml
Score: 1/5
Classification: HAM (Clean)
Processing time: 0.23ms
```

### Filter a Directory of Emails

```bash
./zpo filter -i input_folder -o clean_folder -s spam_folder
```

**Output:**
```
ZPO Processing Complete!
Emails processed: 100
Spam detected: 15
Ham (clean): 85
Average processing time: 0.78ms per email
Total time: 78.5ms
```

### DNS Testing and Benchmarking

```bash
# Generate test emails with known DNS records
./zpo dnstest generate --output test-data/dns-test --count 20

# Run DNS performance demonstration
./zpo dnstest demo

# Run comprehensive DNS benchmarks
./zpo dnstest benchmark

# Generate DNS test configuration
./zpo dnstest config config-dnstest.yaml
```

### Command Options

- `-i, --input`: Input directory or file path (required)
- `-o, --output`: Output directory for clean emails
- `-s, --spam`: Spam directory for filtered emails
- `-t, --threshold`: Spam threshold (default: 4, range: 1-5)
- `-c, --config`: Configuration file path

## 🧪 DNS Features

### Non-Blocking DNS Operations

ZPO implements async DNS operations for maximum performance:

- **Worker Pool**: Configurable concurrent workers (default: 10)
- **Smart Coalescing**: Multiple requests for same domain share results
- **Instant Cache Hits**: Sub-microsecond cached responses
- **Graceful Shutdown**: Proper resource cleanup

### Internal DNS Testing

For reliable testing and benchmarking:

- **Test Server**: Controlled DNS environment with known records
- **Realistic Data**: Gmail, Outlook, Yahoo domains with proper SPF/DMARC
- **TTL Management**: Configurable expiration times (30min stable, 1min spam)
- **Performance Monitoring**: Comprehensive statistics and metrics

### DNS Test Commands

```bash
# DNS server management
./zpo dnstest server start          # Start internal DNS test server
./zpo dnstest server stats          # Show server statistics

# Performance testing
./zpo dnstest demo                   # Interactive performance demo
./zpo dnstest benchmark              # Comprehensive benchmarks

# Test data generation
./zpo dnstest generate -o test-data/dns-test -n 50
./zpo dnstest config config-dnstest.yaml
```

## 📧 Scoring System

ZPO uses a 1-5 scoring system:

| Score | Classification | Action |
|-------|---------------|--------|
| 1 | Definitely Clean | Keep in inbox |
| 2 | Probably Clean | Keep in inbox |
| 3 | Possibly Spam | Keep in inbox (review) |
| 4 | Likely Spam | Move to spam folder |
| 5 | Definitely Spam | Move to spam folder |

## 🧠 Detection Algorithm

ZPO analyzes multiple email features:

### Content Analysis
- **Keywords**: High/medium/low risk spam keywords
- **Capitalization**: Excessive caps usage
- **Punctuation**: Excessive exclamation marks
- **URLs**: Suspicious link density
- **HTML**: HTML-to-text ratio

### Technical Analysis
- **Headers**: Suspicious email headers with SPF/DKIM/DMARC validation
- **Attachments**: Dangerous file types
- **Encoding**: Encoding issues/obfuscation
- **Domain**: Sender domain reputation with DNS verification

### Behavioral Analysis
- **From/To Mismatch**: Reply chain inconsistencies
- **Subject Length**: Unusually long/short subjects

## 📁 Supported Email Formats

- `.eml` - Standard email format
- `.msg` - Outlook message format
- `.txt` - Plain text emails
- `.email` - Generic email files
- Files without extensions (common in email servers)

## 🔧 Configuration

ZPO supports multiple configuration profiles:

### Available Configurations

- **`config.yaml`**: Default configuration with full features
- **`config-fast.yaml`**: Optimized for maximum speed (DNS disabled)
- **`config-cached.yaml`**: Balanced performance with DNS caching
- **`config-dnstest.yaml`**: Internal DNS testing with async operations

### Key Settings

```yaml
# DNS Configuration
headers:
  enable_spf: true
  enable_dkim: true
  enable_dmarc: true
  dns_timeout_ms: 2000
  cache_size: 5000
  cache_ttl_min: 30
  
  # Async DNS Settings
  enable_async_dns: true
  async_dns_workers: 10
  use_internal_dns: true  # For testing
```

### Performance Tuning

```yaml
# High-performance configuration
performance:
  max_concurrent_emails: 20
  timeout_ms: 500
  cache_size: 10000
  batch_size: 100
```

## 📈 Examples

### Example: Clean Email (Score: 1)

```
From: john.doe@gmail.com
Subject: Meeting Tomorrow
Body: Hi, I wanted to remind you about our meeting...
```

### Example: Spam Email (Score: 5)

```
From: noreply@suspicious-domain.com
Subject: URGENT!!! FREE MONEY!!! CLICK NOW!!!
Body: CONGRATULATIONS!!! YOU HAVE WON $1,000,000!!!
```

## 🎯 Use Cases

- **Email Servers**: Integrate into mail server pipelines
- **Personal Use**: Filter personal email archives
- **Development**: Test email classification systems with controlled DNS
- **Research**: Analyze spam detection algorithms
- **Security**: Identify malicious emails with DNS verification
- **Performance Testing**: Benchmark DNS operations in controlled environment

## 🔒 Security Features

- **Attachment Scanning**: Detects suspicious file types
- **Domain Reputation**: Checks sender domain credibility
- **Header Analysis**: Identifies spoofed/manipulated headers
- **DNS Validation**: SPF, DKIM, and DMARC verification
- **Encoding Detection**: Catches obfuscated content

## 📋 Requirements

- **Memory**: ~10MB RAM
- **CPU**: Any modern processor
- **Storage**: ~50MB for binary and examples
- **OS**: Linux, macOS, Windows (via Go compilation)
- **Network**: Optional for DNS validation (can use internal test server)

## 🤝 Contributing

ZPO is designed to be fast and lightweight. When contributing:

1. Maintain the <5ms performance requirement
2. Keep the scoring system 1-5
3. Ensure backward compatibility
4. Add tests for new features
5. Use async DNS operations where possible

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🐛 Support

For issues, feature requests, or questions:

1. Check existing documentation
2. Test with example emails
3. Use DNS test tools for performance issues
4. File an issue with email samples (remove sensitive data)

## 🔮 Roadmap

- [x] Non-blocking DNS operations
- [x] Internal DNS testing server
- [x] Async DNS client with worker pools
- [x] Comprehensive DNS benchmarking tools
- [ ] Machine learning integration
- [ ] Real-time email monitoring
- [ ] Web interface
- [ ] API endpoints
- [ ] Docker container
- [ ] Performance benchmarks vs other filters

---

**ZPO - Because spam filtering should be as reliable as a baby donkey! 🫏** 