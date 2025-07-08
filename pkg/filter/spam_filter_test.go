package filter

import (
	"testing"
	"time"

	"github.com/zpam/spam-filter/pkg/email"
)

func TestSpamFilterCreation(t *testing.T) {
	filter := NewSpamFilter()
	if filter == nil {
		t.Fatal("Failed to create spam filter")
	}

	if filter.parser == nil {
		t.Error("Parser not initialized")
	}

	if len(filter.keywords.HighRisk) == 0 {
		t.Error("High risk keywords not loaded")
	}
}

func TestNormalizeScore(t *testing.T) {
	filter := NewSpamFilter()

	testCases := []struct {
		rawScore float64
		expected int
	}{
		{0, 1},  // Definitely clean
		{3, 1},  // Definitely clean
		{5, 2},  // Probably clean
		{10, 3}, // Possibly spam
		{15, 4}, // Likely spam
		{20, 5}, // Definitely spam
		{25, 5}, // Definitely spam
	}

	for _, tc := range testCases {
		result := filter.normalizeScore(tc.rawScore)
		if result != tc.expected {
			t.Errorf("normalizeScore(%.1f) = %d, expected %d", tc.rawScore, result, tc.expected)
		}
	}
}

func TestKeywordScoring(t *testing.T) {
	filter := NewSpamFilter()

	// Test high-risk keyword
	spamSubject := "FREE MONEY GUARANTEED"
	spamBody := "Act now to get rich quick!"
	score := filter.scoreKeywords(spamSubject, spamBody)

	if score <= 0 {
		t.Error("Expected positive score for spam keywords")
	}

	// Test clean content
	cleanSubject := "Meeting tomorrow"
	cleanBody := "Let's discuss the quarterly reports"
	score = filter.scoreKeywords(cleanSubject, cleanBody)

	if score > 1 {
		t.Error("Expected low score for clean keywords")
	}
}

func TestCapsRatioScoring(t *testing.T) {
	filter := NewSpamFilter()

	// Test high caps ratio (should score high)
	score := filter.scoreCapsRatio(0.8, 0.9)
	if score <= 0 {
		t.Error("Expected positive score for high caps ratio")
	}

	// Test normal caps ratio (should score low)
	score = filter.scoreCapsRatio(0.1, 0.05)
	if score > 0 {
		t.Error("Expected zero score for normal caps ratio")
	}
}

func TestPerformance(t *testing.T) {
	filter := NewSpamFilter()

	// Create a mock email for testing
	testEmail := &email.Email{
		From:    "test@example.com",
		Subject: "Test email",
		Body:    "This is a test email body.",
		Headers: make(map[string]string),
		Features: email.EmailFeatures{
			SubjectLength:         10,
			SubjectCapsRatio:      0.1,
			SubjectExclamations:   0,
			BodyLength:            25,
			BodyHTMLRatio:         0.0,
			BodyCapsRatio:         0.1,
			BodyURLCount:          0,
			BodyExclamations:      0,
			SenderDomainReputable: true,
			FromToMismatch:        false,
			AttachmentCount:       0,
			SuspiciousHeaders:     0,
			EncodingIssues:        false,
		},
	}

	// Measure performance
	start := time.Now()
	for i := 0; i < 1000; i++ {
		filter.calculateSpamScore(testEmail)
	}
	duration := time.Since(start)

	avgTime := duration.Nanoseconds() / 1000 / 1e6 // Convert to milliseconds

	// Should be well under 5ms per email
	if avgTime >= 5 {
		t.Errorf("Performance test failed: average time %.2fms >= 5ms", float64(avgTime))
	}

	t.Logf("Performance test passed: average time %.3fms per email", float64(avgTime))
}

func TestSuspiciousAttachment(t *testing.T) {
	filter := NewSpamFilter()

	// Test suspicious attachment
	suspiciousAttachment := email.Attachment{
		Filename:    "virus.exe",
		ContentType: "application/octet-stream",
		Size:        1024,
	}

	if !filter.isSuspiciousAttachment(suspiciousAttachment) {
		t.Error("Expected suspicious attachment to be detected")
	}

	// Test safe attachment
	safeAttachment := email.Attachment{
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        1024,
	}

	if filter.isSuspiciousAttachment(safeAttachment) {
		t.Error("Expected safe attachment to not be flagged")
	}
}

func TestEmailFileDetection(t *testing.T) {
	filter := NewSpamFilter()

	testCases := []struct {
		filename string
		expected bool
	}{
		{"email.eml", true},
		{"message.msg", true},
		{"email.txt", true},
		{"mail.email", true},
		{"emailfile", true}, // No extension
		{"document.pdf", false},
		{"image.jpg", false},
		{"script.exe", false},
	}

	for _, tc := range testCases {
		result := filter.isEmailFile(tc.filename)
		if result != tc.expected {
			t.Errorf("isEmailFile(%s) = %v, expected %v", tc.filename, result, tc.expected)
		}
	}
}
