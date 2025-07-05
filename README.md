# ZPO - Baby Donkey Spam Filter 🫏

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Speed](https://img.shields.io/badge/Speed-%3C5ms_per_email-brightgreen.svg)](#performance)

ZPO is a lightning-fast, free spam filter that processes emails in under 5ms. Named after baby donkey - it's free, fast, and reliable.

## ✨ Features

- **⚡ Ultra-Fast**: Processes emails in under 5ms
- **🎯 Smart Scoring**: Rates emails 1-5 for precise classification
- **📁 Auto-Sorting**: Automatically moves spam (4-5 rating) to spam folder
- **🔍 Deep Analysis**: Analyzes content, headers, attachments, and sender reputation
- **🆓 Completely Free**: No licensing fees or restrictions
- **🚀 Easy to Use**: Simple CLI interface

## 📊 Performance

ZPO consistently achieves sub-5ms processing times:

- **Clean emails**: ~0.2-0.4ms
- **Spam emails**: ~0.5-0.8ms
- **Average**: ~0.78ms per email
- **Batch processing**: Linear scaling with excellent performance

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

### Command Options

- `-i, --input`: Input directory or file path (required)
- `-o, --output`: Output directory for clean emails
- `-s, --spam`: Spam directory for filtered emails
- `-t, --threshold`: Spam threshold (default: 4, range: 1-5)

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
- **Headers**: Suspicious email headers
- **Attachments**: Dangerous file types
- **Encoding**: Encoding issues/obfuscation
- **Domain**: Sender domain reputation

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

ZPO comes with optimized defaults but can be customized by modifying the source code:

- **Keywords**: Update spam keyword lists in `pkg/filter/spam_filter.go`
- **Weights**: Adjust feature weights for different emphasis
- **Thresholds**: Modify scoring thresholds for different sensitivity

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
- **Development**: Test email classification systems
- **Research**: Analyze spam detection algorithms
- **Security**: Identify malicious emails

## 🔒 Security Features

- **Attachment Scanning**: Detects suspicious file types
- **Domain Reputation**: Checks sender domain credibility
- **Header Analysis**: Identifies spoofed/manipulated headers
- **Encoding Detection**: Catches obfuscated content

## 📋 Requirements

- **Memory**: ~10MB RAM
- **CPU**: Any modern processor
- **Storage**: ~50MB for binary and examples
- **OS**: Linux, macOS, Windows (via Go compilation)

## 🤝 Contributing

ZPO is designed to be fast and lightweight. When contributing:

1. Maintain the <5ms performance requirement
2. Keep the scoring system 1-5
3. Ensure backward compatibility
4. Add tests for new features

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🐛 Support

For issues, feature requests, or questions:

1. Check existing documentation
2. Test with example emails
3. File an issue with email samples (remove sensitive data)

## 🔮 Roadmap

- [ ] Machine learning integration
- [ ] Real-time email monitoring
- [ ] Web interface
- [ ] API endpoints
- [ ] Docker container
- [ ] Performance benchmarks vs other filters

---

**ZPO - Because spam filtering should be as reliable as a baby donkey! 🫏** 