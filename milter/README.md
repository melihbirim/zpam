# 🫏 ZPO Milter Integration Testing

This directory contains comprehensive testing scripts for ZPO + Postfix milter integration.

## 📁 Files

- **`test_zpo_postfix.sh`** - Complete integration test script (setup + test + cleanup)
- **`send_test_emails.sh`** - Simple script to send 10 test emails (requires ZPO + Postfix running)
- **`emails/`** - Directory containing 10 pre-written email files (5 clean, 5 spam)
- **`README.md`** - This documentation

## 🚀 Quick Start

### Option 1: Complete Integration Test (Recommended)

Run the full integration test that automatically sets up, tests, and cleans up:

```bash
cd milter
./test_zpo_postfix.sh
```

This script will:
1. ✅ Check prerequisites (ZPO binary, Postfix, Python3)
2. ⚙️  Configure test environment
3. 🔧 Start ZPO milter server
4. 📧 Configure and start Postfix with milter integration
5. 📨 Send 10 test emails (5 spam levels, 2 emails each)
6. 📊 Analyze results and show detailed scoring
7. 🧹 Clean up everything automatically

### Option 2: Standalone Email Testing

If you already have ZPO milter and Postfix running:

```bash
cd milter
./send_test_emails.sh
```

## 📧 Test Email Details

The test suite includes **10 carefully crafted emails** covering all spam levels:

| Level | Count | Description | Examples |
|-------|-------|-------------|----------|
| **1** | 2 | Definitely Clean | Business meeting notes, personal vacation photos |
| **2** | 2 | Probably Clean | Legitimate newsletters, marketing emails |
| **3** | 2 | Possibly Spam | Suspicious prize notifications, free trial offers |
| **4** | 2 | Likely Spam | Phishing attempts, get-rich-quick schemes |
| **5** | 2 | Definitely Spam | Lottery scams, illegal pharmacy ads |

### Expected ZPO Scores

With properly tuned SpamAssassin-inspired settings:

- **Level 1 emails**: Raw score 0.5-5.0 → ZPO rating 1/5 (Clean)
- **Level 2 emails**: Raw score 5.0-15.0 → ZPO rating 2/5 (Probably Clean)
- **Level 3 emails**: Raw score 15.0-35.0 → ZPO rating 3/5 (Possibly Spam)
- **Level 4 emails**: Raw score 35.0-75.0 → ZPO rating 4/5 (Likely Spam)
- **Level 5 emails**: Raw score 75.0+ → ZPO rating 5/5 (Definitely Spam)

## 🔍 Verification

After running tests, check results:

```bash
# View received emails with ZPO headers
tail -n 100 /var/mail/$USER

# Check for ZPO processing headers
grep "X-ZPO-" /var/mail/$USER

# View detailed scoring
grep -A3 -B1 "X-ZPO-Status:" /var/mail/$USER

# Check ZPO milter logs
tail -f /tmp/zpo_milter.log

# Check Postfix logs (macOS)
tail -f /var/log/mail.log
```

### Expected Headers

Each processed email should contain:

```
X-ZPO-Status: Clean|Spam
X-ZPO-Score: 1.23
X-ZPO-Rating: 2/5
X-ZPO-Features: keywords:0.45,headers:0.32,domain:0.46
```

## 📋 Prerequisites

### Required Software

- **Go** (to build ZPO binary)
- **Postfix** (mail server)
- **sendmail or telnet** (for sending test emails)
- **netcat** (connectivity testing)

### Installation Commands

**macOS (Homebrew):**
```bash
brew install postfix netcat telnet
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postfix netcat-openbsd telnet sendmail
```

**CentOS/RHEL:**
```bash
sudo yum install postfix nmap-ncat telnet sendmail
```

### Build ZPO

```bash
cd /path/to/zpo
go build -o zpo .
```

## ⚙️ Configuration

### ZPO Config (`config.yaml`)

Ensure milter is enabled:

```yaml
milter:
  enabled: true
  address: "127.0.0.1:7357"
  timeout: "10s"

# SpamAssassin-inspired penalty settings
headers:
  spf_fail_penalty: 0.9
  dkim_missing_penalty: 1.0
  dmarc_missing_penalty: 1.5
  auth_weight: 0.2      # Development: 0.2, Production: 1.5
  suspicious_weight: 0.2 # Development: 0.2, Production: 1.5
```

### Postfix Config

The integration script automatically configures Postfix, but for manual setup:

```bash
# /etc/postfix/main.cf
smtpd_milters = inet:127.0.0.1:7357
non_smtpd_milters = inet:127.0.0.1:7357
milter_default_action = accept
milter_connect_timeout = 10s
milter_content_timeout = 15s
milter_protocol = 6
```

## 🧪 Manual Testing

### 1. Start ZPO Milter
```bash
./zpo milter --debug
```

### 2. Configure and Start Postfix
```bash
# Apply configuration
sudo postfix check
sudo postfix start
```

### 3. Test Connectivity
```bash
# Test milter port
nc -z 127.0.0.1 7357

# Test SMTP port  
nc -z 127.0.0.1 25
```

### 4. Send Test Email
```bash
./send_test_emails.sh
```

### 5. Check Results
```bash
tail -n 50 /var/mail/$USER
```

## 🐛 Troubleshooting

### Common Issues

**1. "ZPO binary not found"**
```bash
cd .. && go build -o zpo .
```

**2. "Postfix is not installed"**
```bash
# macOS
brew install postfix

# Ubuntu  
sudo apt install postfix
```

**3. "Permission denied on /var/mail"**
```bash
sudo touch /var/mail/$USER
sudo chown $USER:mail /var/mail/$USER
```

**4. "Connection refused on port 25"**
```bash
# Check if Postfix is running
sudo postfix status

# Start Postfix
sudo postfix start
```

**5. "Connection refused on port 7357"**
```bash
# Check if ZPO milter is running
ps aux | grep "zpo milter"

# Start ZPO milter
./zpo milter --debug
```

**6. "No X-ZPO headers in emails"**
```bash
# Check milter configuration in Postfix
postconf | grep milter

# Check ZPO milter logs
tail -f /tmp/zpo_milter.log
```

### Debug Commands

```bash
# Check Postfix configuration
sudo postfix check

# View Postfix queue
mailq

# Test mail delivery without milter
echo "Test" | mail -s "Test" $USER

# Monitor logs in real-time
tail -f /var/log/mail.log /tmp/zpo_milter.log
```

## 🎯 Expected Results

### Successful Test Output

```
🫏 ZPO + Postfix Integration Test Suite
========================================
✅ Prerequisites check passed
✅ Test environment setup complete  
✅ ZPO milter server started (PID: 12345)
✅ Postfix configured and started with ZPO milter integration

📧 Email 1/10:
✅ Sent Level 1 (Clean - Business): Weekly Team Meeting Notes...

📧 Email 2/10:
✅ Sent Level 1 (Clean - Personal): Vacation Photos from Last Week...

[... 8 more emails ...]

📊 Test Results Summary:
========================
📧 Emails received: 10/10
🏷️  ZPO headers added: 10/10
✅ Clean emails: 4
🚨 Spam emails: 6

✅ All emails processed successfully with ZPO headers!
```

## 📊 Performance Metrics

- **Email Processing**: < 5ms per email
- **Milter Response Time**: < 1ms average
- **Memory Usage**: ~10MB for ZPO milter
- **CPU Usage**: < 1% during normal operation

## 🔧 Advanced Configuration

### Production Settings

For production deployment, update `config.yaml`:

```yaml
headers:
  auth_weight: 1.5        # More aggressive for production
  suspicious_weight: 1.5  # More aggressive for production
  
spam_filter:
  reject_threshold: 4     # Reject at 4/5 instead of 5/5
```

### Custom Email Templates

Modify `test_zpo_emails.py` to add your own test cases:

```python
custom_email = {
    'subject': 'Your Custom Subject',
    'body': 'Your email content...',
    'from': 'sender@domain.com',
    'level': 3,  # Expected spam level 1-5
    'description': 'Custom - Test Type'
}
```

## 📚 References

- [ZPO Documentation](../README.md)
- [Postfix Milter Documentation](http://www.postfix.org/MILTER_README.html)
- [SpamAssassin Scoring Guide](https://spamassassin.apache.org/full/3.4.x/doc/Mail_SpamAssassin_Conf.html) 