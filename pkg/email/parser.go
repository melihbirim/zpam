package email

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"strings"
	"time"
)

// Email represents a parsed email with spam-relevant features
type Email struct {
	From        string
	To          []string
	Subject     string
	Body        string
	Headers     map[string]string
	Attachments []Attachment
	ParsedAt    time.Time
	
	// Spam detection features
	Features EmailFeatures
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
}

// EmailFeatures contains features used for spam detection
type EmailFeatures struct {
	// Header analysis
	SubjectLength     int
	SubjectCapsRatio  float64
	SubjectExclamations int
	
	// Body analysis
	BodyLength        int
	BodyHTMLRatio     float64
	BodyCapsRatio     float64
	BodyURLCount      int
	BodyExclamations  int
	
	// Sender analysis
	SenderDomainReputable bool
	FromToMismatch       bool
	
	// Technical features
	AttachmentCount      int
	SuspiciousHeaders    int
	EncodingIssues       bool
}

// Parser handles fast email parsing
type Parser struct{}

// NewParser creates a new email parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFromFile parses an email from a file
func (p *Parser) ParseFromFile(filepath string) (*Email, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()
	
	return p.Parse(file)
}

// Parse parses an email from a reader
func (p *Parser) Parse(reader io.Reader) (*Email, error) {
	// Parse the email using Go's mail package
	msg, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email: %v", err)
	}
	
	email := &Email{
		Headers:  make(map[string]string),
		ParsedAt: time.Now(),
	}
	
	// Extract basic headers
	email.From = msg.Header.Get("From")
	email.Subject = msg.Header.Get("Subject")
	
	// Extract To addresses
	if to := msg.Header.Get("To"); to != "" {
		email.To = strings.Split(to, ",")
		for i := range email.To {
			email.To[i] = strings.TrimSpace(email.To[i])
		}
	}
	
	// Store all headers for analysis
	for key, values := range msg.Header {
		email.Headers[key] = strings.Join(values, "; ")
	}
	
	// Parse body and attachments
	err = p.parseBodyAndAttachments(msg, email)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body: %v", err)
	}
	
	// Extract features for spam detection
	p.extractFeatures(email)
	
	return email, nil
}

// parseBodyAndAttachments extracts body content and attachments
func (p *Parser) parseBodyAndAttachments(msg *mail.Message, email *Email) error {
	contentType := msg.Header.Get("Content-Type")
	
	if contentType == "" {
		// Simple text email
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return err
		}
		email.Body = string(body)
		return nil
	}
	
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Fallback to reading as plain text
		body, err := io.ReadAll(msg.Body)
		if err != nil {
			return err
		}
		email.Body = string(body)
		return nil
	}
	
	if strings.HasPrefix(mediaType, "multipart/") {
		return p.parseMultipart(msg.Body, params["boundary"], email)
	}
	
	// Single part message
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return err
	}
	email.Body = string(body)
	
	return nil
}

// parseMultipart handles multipart messages
func (p *Parser) parseMultipart(body io.Reader, boundary string, email *Email) error {
	if boundary == "" {
		return fmt.Errorf("multipart message without boundary")
	}
	
	multipartReader := multipart.NewReader(body, boundary)
	
	for {
		part, err := multipartReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		contentType := part.Header.Get("Content-Type")
		contentDisposition := part.Header.Get("Content-Disposition")
		
		if strings.Contains(contentDisposition, "attachment") {
			// Handle attachment
			attachment := Attachment{
				Filename:    part.FileName(),
				ContentType: contentType,
			}
			
			// Read attachment size (for spam detection)
			content, err := io.ReadAll(part)
			if err == nil {
				attachment.Size = int64(len(content))
			}
			
			email.Attachments = append(email.Attachments, attachment)
		} else if strings.HasPrefix(contentType, "text/") {
			// Handle text content
			content, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			
			if email.Body == "" {
				email.Body = string(content)
			} else {
				email.Body += "\n" + string(content)
			}
		}
		
		part.Close()
	}
	
	return nil
}

// extractFeatures calculates spam detection features
func (p *Parser) extractFeatures(email *Email) {
	features := &email.Features
	
	// Subject analysis
	features.SubjectLength = len(email.Subject)
	features.SubjectCapsRatio = p.calculateCapsRatio(email.Subject)
	features.SubjectExclamations = strings.Count(email.Subject, "!")
	
	// Body analysis
	features.BodyLength = len(email.Body)
	features.BodyHTMLRatio = p.calculateHTMLRatio(email.Body)
	features.BodyCapsRatio = p.calculateCapsRatio(email.Body)
	features.BodyURLCount = p.countURLs(email.Body)
	features.BodyExclamations = strings.Count(email.Body, "!")
	
	// Sender analysis
	features.SenderDomainReputable = p.checkDomainReputation(email.From)
	features.FromToMismatch = p.checkFromToMismatch(email)
	
	// Technical features
	features.AttachmentCount = len(email.Attachments)
	features.SuspiciousHeaders = p.countSuspiciousHeaders(email.Headers)
	features.EncodingIssues = p.checkEncodingIssues(email)
}

// calculateCapsRatio calculates the ratio of uppercase letters
func (p *Parser) calculateCapsRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	
	letters := 0
	caps := 0
	
	for _, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			letters++
			if r >= 'A' && r <= 'Z' {
				caps++
			}
		}
	}
	
	if letters == 0 {
		return 0
	}
	
	return float64(caps) / float64(letters)
}

// calculateHTMLRatio calculates the ratio of HTML content
func (p *Parser) calculateHTMLRatio(body string) float64 {
	htmlTags := strings.Count(strings.ToLower(body), "<") + 
		strings.Count(strings.ToLower(body), ">")
	
	if len(body) == 0 {
		return 0
	}
	
	return float64(htmlTags) / float64(len(body))
}

// countURLs counts URLs in the text
func (p *Parser) countURLs(text string) int {
	count := 0
	count += strings.Count(strings.ToLower(text), "http://")
	count += strings.Count(strings.ToLower(text), "https://")
	count += strings.Count(strings.ToLower(text), "www.")
	return count
}

// checkDomainReputation checks if sender domain is reputable
func (p *Parser) checkDomainReputation(from string) bool {
	// Simple reputation check - in production this would use a reputation service
	from = strings.ToLower(from)
	
	reputableDomains := []string{
		"gmail.com", "yahoo.com", "outlook.com", "hotmail.com",
		"apple.com", "microsoft.com", "google.com", "amazon.com",
	}
	
	for _, domain := range reputableDomains {
		if strings.Contains(from, domain) {
			return true
		}
	}
	
	return false
}

// checkFromToMismatch checks for From/To field mismatches
func (p *Parser) checkFromToMismatch(email *Email) bool {
	// Simplified check - in production this would be more sophisticated
	return strings.Contains(strings.ToLower(email.Subject), "re:") && 
		len(email.To) == 0
}

// countSuspiciousHeaders counts suspicious email headers
func (p *Parser) countSuspiciousHeaders(headers map[string]string) int {
	suspicious := 0
	
	suspiciousHeaders := []string{
		"X-Spam", "X-Bulk", "X-Advertisement", "X-Mailer-Version",
		"X-Priority", "X-Msmail-Priority",
	}
	
	for header := range headers {
		headerLower := strings.ToLower(header)
		for _, suspicious_header := range suspiciousHeaders {
			if strings.Contains(headerLower, strings.ToLower(suspicious_header)) {
				suspicious++
				break
			}
		}
	}
	
	return suspicious
}

// checkEncodingIssues checks for encoding problems
func (p *Parser) checkEncodingIssues(email *Email) bool {
	// Simple check for common encoding issues
	text := email.Subject + email.Body
	return strings.Contains(text, "?=") || 
		strings.Contains(text, "=?") ||
		strings.Count(text, "�") > 0
} 