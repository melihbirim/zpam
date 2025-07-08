#!/bin/bash

# 🫏 ZPAM Email Sender Script
# Simple script to send 10 test emails (5 clean, 5 spam) to test ZPAM milter

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

EMAILS_DIR="../training-data"
SUCCESSFUL_SENDS=0

echo -e "${BLUE}🫏 ZPAM Email Test Suite - 10 Emails (5 Clean, 5 Spam)${NC}"
echo "====================================================="

# Check prerequisites
if [[ ! -d "$EMAILS_DIR" ]]; then
    echo -e "${RED}❌ Training data directory '$EMAILS_DIR' not found${NC}"
    echo "Make sure you're running from the milter directory."
    exit 1
fi

if [[ ! -d "$EMAILS_DIR/spam" ]] || [[ ! -d "$EMAILS_DIR/ham" ]]; then
    echo -e "${RED}❌ Training data structure incomplete. Expected spam/ and ham/ subdirectories.${NC}"
    exit 1
fi

if ! command -v sendmail &> /dev/null && ! command -v telnet &> /dev/null; then
    echo -e "${RED}❌ Neither sendmail nor telnet is available${NC}"
    echo "Install one of them for email testing."
    exit 1
fi

# Send emails function
send_email_file() {
    local email_file="$1"
    local description="$2"
    local subject="$3"
    
    # Try sendmail first
    if command -v sendmail &> /dev/null; then
        if sendmail -t < "$email_file" 2>/dev/null; then
            echo -e "✅ Sent ($description): $subject..."
            return 0
        fi
    fi
    
    # Fallback to telnet
    local from_addr=$(grep "^From:" "$email_file" | cut -d' ' -f2-)
    local to_addr=$(grep "^To:" "$email_file" | cut -d' ' -f2-)
    local email_content=$(cat "$email_file")
    
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
        echo -e "✅ Sent ($description): $subject..."
        return 0
    else
        echo -e "${RED}❌ Failed ($description): Could not send email${NC}"
        return 1
    fi
}

# Email descriptions
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

# Send each email
count=1
for email_file in $(ls "$EMAILS_DIR"/ham/*.eml "$EMAILS_DIR"/spam/*.eml | sort); do
    filename=$(basename "$email_file")
    description="${email_descriptions[$filename]:-Unknown}"
    subject=$(grep "^Subject:" "$email_file" | cut -d' ' -f2- | head -c50)
    
    echo
    echo "📧 Email $count/10:"
    
    if send_email_file "$email_file" "$description" "$subject"; then
        SUCCESSFUL_SENDS=$((SUCCESSFUL_SENDS + 1))
    fi
    
    count=$((count + 1))
    sleep 1
done

echo
echo -e "${BLUE}📊 Test Summary:${NC}"
echo "   Emails sent: $SUCCESSFUL_SENDS/10"
echo "   Clean emails: 5 (01-05)"
echo "   Spam emails: 5 (06-10)"
echo
echo -e "${GREEN}🔍 Check /var/mail/\$USER for X-ZPAM-* headers${NC}"

if [[ $SUCCESSFUL_SENDS -eq 10 ]]; then
    echo -e "${GREEN}🎉 All emails sent successfully!${NC}"
else
    echo -e "${RED}⚠️  Only $SUCCESSFUL_SENDS/10 emails sent successfully${NC}"
fi 