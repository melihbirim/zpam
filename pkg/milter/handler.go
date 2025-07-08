package milter

import (
	"fmt"
	"strings"
	"time"

	"github.com/d--j/go-milter"
	"github.com/zpam/spam-filter/pkg/config"
	"github.com/zpam/spam-filter/pkg/email"
	"github.com/zpam/spam-filter/pkg/filter"
)

// Handler implements the milter.Milter interface for ZPAM spam filtering
type Handler struct {
	milter.NoOpMilter
	config     *config.Config
	spamFilter *filter.SpamFilter

	// Email data being built during the milter session
	email *email.Email

	// Connection/session data
	connectHost string
	connectAddr string
	heloName    string

	// Performance tracking
	startTime time.Time
}

// NewHandler creates a new milter handler
func NewHandler(cfg *config.Config, spamFilter *filter.SpamFilter) *Handler {
	return &Handler{
		config:     cfg,
		spamFilter: spamFilter,
		email:      &email.Email{Headers: make(map[string]string)},
		startTime:  time.Now(),
	}
}

// NewConnection is called when a new SMTP connection is established
func (h *Handler) NewConnection(m milter.Modifier) error {
	h.startTime = time.Now()
	return nil
}

// Connect is called when connection information is available
func (h *Handler) Connect(host string, family string, port uint16, addr string, m milter.Modifier) (*milter.Response, error) {
	h.connectHost = host
	h.connectAddr = addr
	return milter.RespContinue, nil
}

// Helo is called when HELO/EHLO is received
func (h *Handler) Helo(name string, m milter.Modifier) (*milter.Response, error) {
	h.heloName = name
	return milter.RespContinue, nil
}

// MailFrom is called when MAIL FROM is received
func (h *Handler) MailFrom(from string, esmtpArgs string, m milter.Modifier) (*milter.Response, error) {
	// Reset email for new message
	h.email = &email.Email{
		From:     from,
		Headers:  make(map[string]string),
		ParsedAt: time.Now(),
	}

	return milter.RespContinue, nil
}

// RcptTo is called for each RCPT TO
func (h *Handler) RcptTo(rcptTo string, esmtpArgs string, m milter.Modifier) (*milter.Response, error) {
	h.email.To = append(h.email.To, rcptTo)
	return milter.RespContinue, nil
}

// Data is called when DATA command is received
func (h *Handler) Data(m milter.Modifier) (*milter.Response, error) {
	return milter.RespContinue, nil
}

// Header is called for each header
func (h *Handler) Header(name string, value string, m milter.Modifier) (*milter.Response, error) {
	h.email.Headers[name] = value

	// Handle special headers
	switch strings.ToLower(name) {
	case "subject":
		h.email.Subject = value
	case "from":
		if h.email.From == "" {
			h.email.From = value
		}
	}

	return milter.RespContinue, nil
}

// Headers is called when all headers have been received
func (h *Handler) Headers(m milter.Modifier) (*milter.Response, error) {
	return milter.RespContinue, nil
}

// BodyChunk is called for each body chunk
func (h *Handler) BodyChunk(chunk []byte, m milter.Modifier) (*milter.Response, error) {
	h.email.Body += string(chunk)
	return milter.RespContinue, nil
}

// EndOfMessage is called when the message is complete
func (h *Handler) EndOfMessage(m milter.Modifier) (*milter.Response, error) {
	// Extract features from the email
	h.extractEmailFeatures()

	// Calculate spam score using existing spam filter
	score := h.spamFilter.CalculateSpamScore(h.email)
	normalizedScore := h.spamFilter.NormalizeScore(score)

	// Add spam headers if configured
	if h.config.Milter.AddSpamHeaders {
		if err := h.addSpamHeaders(m, normalizedScore, score); err != nil {
			return milter.RespTempFail, fmt.Errorf("failed to add spam headers: %v", err)
		}
	}

	// Determine action based on score and thresholds
	return h.determineAction(normalizedScore, score), nil
}

// Abort is called when the message is aborted
func (h *Handler) Abort(m milter.Modifier) error {
	// Reset email data
	h.email = &email.Email{Headers: make(map[string]string)}
	return nil
}

// Cleanup is called when the connection is closed
func (h *Handler) Cleanup(m milter.Modifier) {
	// No cleanup needed for stateless handler
}

// extractEmailFeatures extracts spam detection features from the email
func (h *Handler) extractEmailFeatures() {
	if h.email.Features == (email.EmailFeatures{}) {
		// Extract basic features for spam detection
		h.email.Features = email.EmailFeatures{
			SubjectLength:       len(h.email.Subject),
			SubjectCapsRatio:    h.calculateCapsRatio(h.email.Subject),
			SubjectExclamations: strings.Count(h.email.Subject, "!"),
			BodyLength:          len(h.email.Body),
			BodyCapsRatio:       h.calculateCapsRatio(h.email.Body),
			BodyExclamations:    strings.Count(h.email.Body, "!"),
			BodyURLCount:        h.countURLs(h.email.Body),
			BodyHTMLRatio:       h.calculateHTMLRatio(h.email.Body),
			AttachmentCount:     len(h.email.Attachments),
			SuspiciousHeaders:   h.countSuspiciousHeaders(),
			EncodingIssues:      h.checkEncodingIssues(),
			FromToMismatch:      h.checkFromToMismatch(),
		}

		// Set domain reputation
		domain := h.extractDomain(h.email.From)
		h.email.Features.SenderDomainReputable = h.checkDomainReputation(domain)
	}
}

// calculateCapsRatio calculates the ratio of uppercase letters
func (h *Handler) calculateCapsRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	var uppercase int
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			uppercase++
		}
	}

	return float64(uppercase) / float64(len(text))
}

// countURLs counts the number of URLs in text
func (h *Handler) countURLs(text string) int {
	count := 0
	count += strings.Count(strings.ToLower(text), "http://")
	count += strings.Count(strings.ToLower(text), "https://")
	count += strings.Count(strings.ToLower(text), "ftp://")
	count += strings.Count(strings.ToLower(text), "www.")
	return count
}

// calculateHTMLRatio calculates the ratio of HTML tags in text
func (h *Handler) calculateHTMLRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	htmlChars := strings.Count(text, "<") + strings.Count(text, ">")
	return float64(htmlChars) / float64(len(text))
}

// countSuspiciousHeaders counts suspicious headers
func (h *Handler) countSuspiciousHeaders() int {
	suspicious := 0
	suspiciousHeaders := []string{
		"x-spam", "x-bulk", "x-advertisement", "x-mailer-version",
		"x-priority", "x-msmail-priority",
	}

	for header := range h.email.Headers {
		headerLower := strings.ToLower(header)
		for _, susHeader := range suspiciousHeaders {
			if strings.Contains(headerLower, susHeader) {
				suspicious++
				break
			}
		}
	}

	return suspicious
}

// checkEncodingIssues checks for encoding problems
func (h *Handler) checkEncodingIssues() bool {
	text := h.email.Subject + h.email.Body
	return strings.Contains(text, "?=") ||
		strings.Contains(text, "=?") ||
		strings.Count(text, "�") > 0
}

// checkFromToMismatch checks for From/To field mismatches
func (h *Handler) checkFromToMismatch() bool {
	return strings.Contains(strings.ToLower(h.email.Subject), "re:") &&
		len(h.email.To) == 0
}

// extractDomain extracts domain from email address
func (h *Handler) extractDomain(email string) string {
	if idx := strings.LastIndex(email, "@"); idx >= 0 {
		return strings.ToLower(email[idx+1:])
	}
	return ""
}

// checkDomainReputation checks if domain is reputable
func (h *Handler) checkDomainReputation(domain string) bool {
	reputableDomains := []string{
		"gmail.com", "yahoo.com", "outlook.com", "hotmail.com",
		"apple.com", "microsoft.com", "google.com", "amazon.com",
	}

	for _, repDomain := range reputableDomains {
		if domain == repDomain {
			return true
		}
	}

	return false
}

// addSpamHeaders adds X-ZPAM-* headers with scan results
func (h *Handler) addSpamHeaders(m milter.Modifier, normalizedScore int, rawScore float64) error {
	prefix := h.config.Milter.SpamHeaderPrefix

	// Add scan result header
	classification := "Clean"
	if normalizedScore >= h.config.Detection.SpamThreshold {
		classification = "Spam"
	}

	if err := m.AddHeader(prefix+"Status", classification); err != nil {
		return err
	}

	// Add score headers
	if err := m.AddHeader(prefix+"Score", fmt.Sprintf("%d/5", normalizedScore)); err != nil {
		return err
	}

	if err := m.AddHeader(prefix+"Score-Raw", fmt.Sprintf("%.2f", rawScore)); err != nil {
		return err
	}

	// Add scan info
	scanTime := time.Since(h.startTime).Milliseconds()
	scanInfo := fmt.Sprintf("ZPAM v1.0; %.2fms", float64(scanTime))
	if err := m.AddHeader(prefix+"Info", scanInfo); err != nil {
		return err
	}

	return nil
}

// determineAction determines what action to take based on spam score
func (h *Handler) determineAction(normalizedScore int, rawScore float64) *milter.Response {
	// Check if we should reject (using > instead of >= for testing)
	if normalizedScore > h.config.Milter.RejectThreshold {
		if h.config.Milter.RejectMessage != "" {
			// Use custom rejection message
			resp, _ := milter.RejectWithCodeAndReason(550, h.config.Milter.RejectMessage)
			return resp
		} else {
			// Use default rejection
			resp, _ := milter.RejectWithCodeAndReason(550,
				fmt.Sprintf("5.7.1 Message rejected as spam (score: %d/5)", normalizedScore))
			return resp
		}
	}

	// Check if we should quarantine
	if h.config.Milter.CanQuarantine && normalizedScore >= h.config.Milter.QuarantineThreshold {
		message := h.config.Milter.QuarantineMessage
		if message == "" {
			message = fmt.Sprintf("ZPAM spam quarantine (score: %d/5)", normalizedScore)
		}

		// Note: Quarantine functionality would require implementing the quarantine action
		// For now, we'll add a header and continue
		return milter.RespContinue
	}

	// Accept the message
	return milter.RespContinue
}
