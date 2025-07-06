#!/bin/bash

# 🫏 ZPO + Postfix Integration Test Script
# This script sets up, tests, and tears down the complete milter integration

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
ZPO_BINARY="../zpo"
CONFIG_FILE="../config.yaml"
MILTER_PORT="7357"
POSTFIX_TEST_CONFIG="/tmp/postfix_main.cf.zpo_test"
PYTHON_TEST_SCRIPT="temp_test_emails.py"
TEST_USER="$USER"
MAIL_FILE="/var/mail/$TEST_USER"

# Helper functions
log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if ZPO binary exists
    if [[ ! -f "$ZPO_BINARY" ]]; then
        error "ZPO binary not found at $ZPO_BINARY. Run 'go build -o zpo .' first."
    fi
    
    # Check if config file exists
    if [[ ! -f "$CONFIG_FILE" ]]; then
        error "Config file not found at $CONFIG_FILE"
    fi
    
    # Check if postfix is available
    if ! command -v postfix &> /dev/null; then
        error "Postfix is not installed. Install with: brew install postfix (macOS) or apt install postfix (Ubuntu)"
    fi
    
    # Check if sendmail or telnet is available
    if ! command -v sendmail &> /dev/null && ! command -v telnet &> /dev/null; then
        error "Neither sendmail nor telnet is available. Install one of them for email testing."
    fi
    
    # Check if we have the emails directory
    if [[ ! -d "emails" ]]; then
        error "Email files directory 'emails' not found. Make sure you're running from the milter directory."
    fi
    
    success "Prerequisites check passed"
}

# Setup test environment
setup_environment() {
    log "Setting up test environment..."
    
    # Enable milter in config
    if grep -q "enabled: false" "$CONFIG_FILE"; then
        sed -i.bak 's/enabled: false/enabled: true/' "$CONFIG_FILE"
        info "Enabled milter in config.yaml"
    fi
    
    # Backup original postfix config if it exists
    if [[ -f "/etc/postfix/main.cf" ]]; then
        sudo cp /etc/postfix/main.cf /etc/postfix/main.cf.backup.zpo || warning "Could not backup postfix config"
    fi
    
    # Create test postfix configuration
    cat > "$POSTFIX_TEST_CONFIG" << 'EOF'
# ZPO + Postfix Integration Test Configuration
myhostname = localhost
mydomain = localhost
myorigin = $mydomain
inet_interfaces = localhost
mydestination = $myhostname, localhost.$mydomain, localhost, $mydomain
relayhost = 
mynetworks = 127.0.0.0/8

# ZPO Milter Configuration
smtpd_milters = inet:127.0.0.1:7357
non_smtpd_milters = inet:127.0.0.1:7357
milter_default_action = accept
milter_connect_timeout = 10s
milter_content_timeout = 15s
milter_protocol = 6

# Debugging for testing
debug_peer_level = 1
debug_peer_list = 127.0.0.1
EOF
    
    success "Test environment setup complete"
}

# Start ZPO milter server
start_zpo_milter() {
    log "Starting ZPO milter server..."
    
    # Kill any existing ZPO processes
    pkill -f "zpo milter" 2>/dev/null || true
    
    # Start ZPO milter in background  
    $ZPO_BINARY milter --config "$CONFIG_FILE" --debug > zpo_milter.log 2>&1 &
    ZPO_PID=$!
    
    # Wait for milter to start
    sleep 3
    
    # Check if milter is running
    if ! kill -0 $ZPO_PID 2>/dev/null; then
        error "Failed to start ZPO milter. Check zpo_milter.log for details."
    fi
    
    # Test connectivity
    if ! nc -z 127.0.0.1 $MILTER_PORT 2>/dev/null; then
        error "ZPO milter is not listening on port $MILTER_PORT"
    fi
    
    success "ZPO milter server started (PID: $ZPO_PID)"
}

# Configure and start postfix
setup_postfix() {
    log "Configuring and starting Postfix..."
    
    # Stop postfix if running
    sudo postfix stop 2>/dev/null || true
    
    # Apply test configuration
    sudo cp "$POSTFIX_TEST_CONFIG" /etc/postfix/main.cf
    
    # Check configuration
    sudo postfix check || error "Postfix configuration check failed"
    
    # Start postfix
    sudo postfix start || error "Failed to start Postfix"
    
    success "Postfix configured and started with ZPO milter integration"
}

# Run email tests
run_email_tests() {
    log "Running email tests..."
    
    # Clear existing mail
    > "$MAIL_FILE" 2>/dev/null || sudo touch "$MAIL_FILE"
    
    # Send test emails using the pre-written .eml files
    send_test_emails
    
    # Wait for emails to be processed
    sleep 5
    
    # Check results
    check_test_results
}

# Send test emails using pre-written .eml files
send_test_emails() {
    local emails_dir="emails"
    local successful_sends=0
    
    if [[ ! -d "$emails_dir" ]]; then
        error "Email files directory not found: $emails_dir"
    fi
    
    echo "🫏 ZPO Email Test Suite - 10 Emails (5 Clean, 5 Spam)"
    echo "====================================================="
    
    # Define email files with descriptions
    declare -A email_descriptions=(
        ["01_clean_business.eml"]="Clean - Business"
        ["02_clean_personal.eml"]="Clean - Personal"
        ["03_clean_newsletter.eml"]="Clean - Newsletter"
        ["04_clean_marketing.eml"]="Clean - Marketing"
        ["05_clean_update.eml"]="Clean - Update"
        ["06_spam_phishing.eml"]="Spam - Phishing"
        ["07_spam_getrich.eml"]="Spam - Get Rich"
        ["08_spam_lottery.eml"]="Spam - Lottery"
        ["09_spam_drugs.eml"]="Spam - Drugs"
        ["10_spam_prize.eml"]="Spam - Prize"
    )
    
    # Send each email file
    local count=1
    for email_file in $(ls "$emails_dir"/*.eml | sort); do
        local filename=$(basename "$email_file")
        local description="${email_descriptions[$filename]:-Unknown}"
        local subject=$(grep "^Subject:" "$email_file" | cut -d' ' -f2- | head -c50)
        
        echo
        echo "📧 Email $count/10:"
        
        if send_email_file "$email_file" "$description" "$subject"; then
            successful_sends=$((successful_sends + 1))
        fi
        
        count=$((count + 1))
        sleep 1  # Small delay between emails
    done
    
    echo
    echo "📊 Test Summary:"
    echo "   Emails sent: $successful_sends/10"
    echo "   Clean emails: 5 (01-05)"
    echo "   Spam emails: 5 (06-10)"
    echo
    echo "🔍 Check /var/mail/$USER for X-ZPO-* headers"
}

# Send individual email file via sendmail
send_email_file() {
    local email_file="$1"
    local description="$2"
    local subject="$3"
    
    if [[ ! -f "$email_file" ]]; then
        echo "❌ Failed: Email file not found: $email_file"
        return 1
    fi
    
    # Try using sendmail first, then fall back to telnet
    if command -v sendmail &> /dev/null; then
        if sendmail -t < "$email_file" 2>/dev/null; then
            echo "✅ Sent ($description): $subject..."
            return 0
        fi
    fi
    
    # Fallback: send via telnet to SMTP port
    local from_addr=$(grep "^From:" "$email_file" | cut -d' ' -f2-)
    local to_addr=$(grep "^To:" "$email_file" | cut -d' ' -f2-)
    
    if send_via_telnet "$email_file" "$from_addr" "$to_addr" "$description" "$subject"; then
        return 0
    fi
    
    echo "❌ Failed ($description): Could not send email"
    return 1
}

# Send email via telnet to SMTP port (fallback method)
send_via_telnet() {
    local email_file="$1"
    local from_addr="$2"
    local to_addr="$3"
    local description="$4"
    local subject="$5"
    
    # Extract email body (everything after first blank line)
    local email_content=$(cat "$email_file")
    
    # Send via telnet
    (
        echo "HELO localhost"
        echo "MAIL FROM: <$from_addr>"
        echo "RCPT TO: <$to_addr>"
        echo "DATA"
        echo "$email_content"
        echo "."
        echo "QUIT"
        sleep 1
    ) | telnet localhost 25 2>/dev/null 1>/dev/null
    
    if [[ $? -eq 0 ]]; then
        echo "✅ Sent ($description): $subject..."
        return 0
    fi
    
    return 1
}

# Check test results
check_test_results() {
    log "Analyzing test results..."
    
    # Check if emails were received
    if [[ ! -f "$MAIL_FILE" ]]; then
        warning "Mail file not found at $MAIL_FILE"
        return
    fi
    
    # Count emails received
    email_count=$(grep -c "^From " "$MAIL_FILE" 2>/dev/null || echo "0")
    
    # Count X-ZPO headers
    zpo_headers=$(grep -c "X-ZPO-Status:" "$MAIL_FILE" 2>/dev/null || echo "0")
    
    # Check score distribution
    clean_count=$(grep -c "X-ZPO-Status: Clean" "$MAIL_FILE" 2>/dev/null || echo "0")
    spam_count=$(grep -c "X-ZPO-Status: Spam" "$MAIL_FILE" 2>/dev/null || echo "0")
    
    echo
    echo "📊 Test Results Summary:"
    echo "========================"
    echo "📧 Emails received: $email_count/10"
    echo "🏷️  ZPO headers added: $zpo_headers/10"
    echo "✅ Clean emails: $clean_count"
    echo "🚨 Spam emails: $spam_count"
    
    if [[ $email_count -eq 10 && $zpo_headers -eq 10 ]]; then
        success "All emails processed successfully with ZPO headers!"
    elif [[ $email_count -eq 10 ]]; then
        warning "All emails received but missing ZPO headers ($zpo_headers/10)"
    else
        warning "Only $email_count/10 emails received"
    fi
    
    # Show detailed scoring
    echo
    echo "📈 Detailed Scoring Results:"
    echo "============================"
    grep -A2 -B1 "X-ZPO-Status:" "$MAIL_FILE" 2>/dev/null | grep -E "(Subject:|X-ZPO-Status:|X-ZPO-Score:)" || warning "No detailed scores found"
}

# Cleanup function
cleanup() {
    log "Cleaning up test environment..."
    
    # Stop ZPO milter
    if [[ -n "$ZPO_PID" ]]; then
        kill $ZPO_PID 2>/dev/null || true
        success "Stopped ZPO milter"
    fi
    
    # Stop Postfix
    sudo postfix stop 2>/dev/null || true
    success "Stopped Postfix"
    
    # Restore original postfix config if backup exists
    if [[ -f "/etc/postfix/main.cf.backup.zpo" ]]; then
        sudo cp /etc/postfix/main.cf.backup.zpo /etc/postfix/main.cf
        info "Restored original Postfix configuration"
    fi
    
    # Restore original ZPO config if backup exists
    if [[ -f "$CONFIG_FILE.bak" ]]; then
        mv "$CONFIG_FILE.bak" "$CONFIG_FILE"
        info "Restored original ZPO configuration"
    fi
    
    # Clean up temporary files
    rm -f "$POSTFIX_TEST_CONFIG"
    rm -f zpo_milter.log
    
    success "Cleanup complete"
}

# Signal handlers
trap cleanup EXIT
trap 'error "Script interrupted"' INT TERM

# Main execution
main() {
    echo "🫏 ZPO + Postfix Integration Test Suite"
    echo "========================================"
    
    check_prerequisites
    setup_environment
    start_zpo_milter
    setup_postfix
    run_email_tests
    
    echo
    success "Integration test complete!"
    echo
    info "View full results: tail -n 100 $MAIL_FILE"
    info "View ZPO logs: tail -f milter/zpo_milter.log"
    info "View Postfix logs: tail -f /var/log/mail.log"
}

# Run main function
main "$@" 