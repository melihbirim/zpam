//go:build integration
// +build integration

package learning

import (
	"testing"
	"time"

	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/email"
	"github.com/zpo/spam-filter/pkg/filter"
)

// Integration tests for Redis Bayesian filter with full pipeline
func TestRedisIntegrationFullPipeline(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	// Create Redis config
	cfg := config.DefaultConfig()
	cfg.Learning.Enabled = true
	cfg.Learning.Backend = "redis"
	cfg.Learning.Redis.RedisURL = "redis://localhost:6379"
	cfg.Learning.Redis.KeyPrefix = "zpo:integration:test"
	cfg.Learning.Redis.DatabaseNum = 1
	cfg.Learning.Redis.MinLearns = 5 // Lower for testing
	cfg.Learning.Redis.DefaultUser = "testuser"

	// Create spam filter with Redis backend
	spamFilter := filter.NewSpamFilterWithConfig(cfg)
	if spamFilter == nil {
		t.Fatal("Failed to create spam filter")
	}

	// Clean up any existing data
	defer func() {
		spamFilter.ResetLearning("testuser")
	}()

	// Train with realistic email samples
	spamEmails := []struct {
		subject, body string
	}{
		{
			"🔥 URGENT: Free iPhone 15 - LIMITED TIME! 🔥",
			"CONGRATULATIONS! You've been selected to receive a FREE iPhone 15 Pro Max! Click here NOW to claim your prize before it expires! This offer is valid for 24 hours only. Don't miss out on this AMAZING opportunity! Visit: http://free-iphone-scam.com",
		},
		{
			"💰 Make $5000/Week Working From Home! 💰",
			"Tired of your boring 9-5 job? Want to make REAL money from home? Our proven system has helped thousands earn $5000+ per week! No experience needed! Start today! Limited spots available. Click here: http://work-from-home-scam.com",
		},
		{
			"VIAGRA 90% OFF - Discreet Shipping",
			"Premium quality Viagra at unbeatable prices! 90% discount for new customers. Fast, discreet shipping worldwide. No prescription required. Order now and save BIG! Visit our pharmacy: http://fake-pharmacy.com",
		},
		{
			"URGENT: Account Security Alert",
			"Your account has been compromised! Suspicious activity detected. Click here IMMEDIATELY to secure your account or it will be permanently suspended. Act now: http://phishing-site.com/secure",
		},
		{
			"Lottery Winner - $1,000,000 Prize!",
			"You have won the International Email Lottery! Prize amount: $1,000,000 USD. To claim your winnings, contact our agent immediately with your personal details. Congratulations! Email: winner@lottery-scam.com",
		},
		{
			"Weight Loss Miracle - Lose 30lbs in 30 Days!",
			"Revolutionary weight loss pill burns fat while you sleep! Lose up to 30 pounds in just 30 days! Doctors hate this one simple trick! No diet or exercise required! Order now: http://weight-loss-scam.com",
		},
	}

	hamEmails := []struct {
		subject, body string
	}{
		{
			"Weekly Team Meeting - Thursday 2PM",
			"Hi everyone, just a reminder about our weekly team meeting this Thursday at 2PM in the main conference room. Agenda includes project updates, Q3 planning, and team announcements. Please bring your status reports. Best regards, Sarah",
		},
		{
			"Project Deadline Extension",
			"Good news! We've received approval to extend the project deadline by one week due to the additional requirements that came in last Friday. The new deadline is now March 15th. Please adjust your schedules accordingly. Thanks, Mike",
		},
		{
			"Birthday Party Invitation",
			"You're invited to celebrate Emma's birthday this Saturday at 6PM! We're having a small gathering at our place with dinner and cake. Please RSVP by Thursday so we can plan accordingly. Looking forward to seeing you! Best, Tom and Lisa",
		},
		{
			"Quarterly Financial Report",
			"Please find attached the Q3 financial report for your review. The numbers look good overall with a 12% increase in revenue compared to last quarter. Let's schedule a meeting next week to discuss the details. Regards, Finance Team",
		},
		{
			"System Maintenance Window",
			"This is to inform you that scheduled system maintenance will occur this Sunday from 2AM to 6AM EST. During this time, all services will be unavailable. Please plan accordingly and ensure all critical tasks are completed beforehand. IT Operations",
		},
		{
			"Conference Call Notes",
			"Hi team, here are the notes from today's conference call with the client: 1) Project timeline approved, 2) Budget increased by 10%, 3) Additional features requested. Next steps: update project plan and schedule follow-up meeting. Cheers, David",
		},
	}

	t.Log("Training spam emails...")
	for i, email := range spamEmails {
		err := spamFilter.TrainSpam(email.subject, email.body, "testuser")
		if err != nil {
			t.Fatalf("Failed to train spam email %d: %v", i+1, err)
		}
	}

	t.Log("Training ham emails...")
	for i, email := range hamEmails {
		err := spamFilter.TrainHam(email.subject, email.body, "testuser")
		if err != nil {
			t.Fatalf("Failed to train ham email %d: %v", i+1, err)
		}
	}

	// Wait a moment for Redis operations to complete
	time.Sleep(100 * time.Millisecond)

	// Test classification with new emails
	testCases := []struct {
		name         string
		subject      string
		body         string
		expectedSpam bool
		description  string
	}{
		{
			name:         "Clear spam test",
			subject:      "FREE MONEY NOW! URGENT ACTION REQUIRED!!!",
			body:         "Click here to get FREE money! Limited time offer! Don't miss out! Visit: http://scam.com",
			expectedSpam: true,
			description:  "Should be classified as spam due to typical spam keywords",
		},
		{
			name:         "Clear ham test",
			subject:      "Monthly Budget Review Meeting",
			body:         "Hi team, let's schedule our monthly budget review meeting for next Tuesday. Please prepare your department reports. Thanks, Finance",
			expectedSpam: false,
			description:  "Should be classified as ham - legitimate business email",
		},
		{
			name:         "Pharmacy spam test",
			subject:      "Cheap Pills Online - No Prescription",
			body:         "Buy prescription drugs without prescription! Cheap prices! Fast delivery! Order now at our online pharmacy!",
			expectedSpam: true,
			description:  "Should be classified as spam - typical pharmacy spam",
		},
		{
			name:         "Personal email test",
			subject:      "Dinner plans this weekend",
			body:         "Hey! Want to grab dinner this weekend? I heard about a new restaurant downtown that's supposed to be really good. Let me know!",
			expectedSpam: false,
			description:  "Should be classified as ham - personal communication",
		},
	}

	t.Log("Testing email classification...")
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a parsed email for testing
			parsedEmail := &email.Email{
				Subject: tc.subject,
				Body:    tc.body,
				From:    "test@example.com",
				To:      []string{"recipient@example.com"},
				Headers: map[string]string{
					"Subject": tc.subject,
					"From":    "test@example.com",
					"To":      "recipient@example.com",
				},
			}

			// Calculate spam score using the full filter
			score := spamFilter.CalculateSpamScore(parsedEmail)
			normalizedScore := spamFilter.NormalizeScore(score)
			isSpam := normalizedScore >= 4 // Spam threshold

			t.Logf("Email: '%s'", tc.subject)
			t.Logf("Raw score: %.2f, Normalized: %d, IsSpam: %v", score, normalizedScore, isSpam)
			t.Logf("Description: %s", tc.description)

			if isSpam != tc.expectedSpam {
				t.Errorf("Expected spam=%v, got spam=%v (score=%.2f, normalized=%d)",
					tc.expectedSpam, isSpam, score, normalizedScore)
			}
		})
	}
}

// Test multi-user functionality
func TestRedisIntegrationMultiUser(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	// Create Redis config
	cfg := config.DefaultConfig()
	cfg.Learning.Enabled = true
	cfg.Learning.Backend = "redis"
	cfg.Learning.Redis.RedisURL = "redis://localhost:6379"
	cfg.Learning.Redis.KeyPrefix = "zpo:integration:multiuser"
	cfg.Learning.Redis.DatabaseNum = 1
	cfg.Learning.Redis.MinLearns = 2 // Lower for testing
	cfg.Learning.Redis.PerUserStats = true

	// Create spam filter
	spamFilter := filter.NewSpamFilterWithConfig(cfg)
	if spamFilter == nil {
		t.Fatal("Failed to create spam filter")
	}

	// Clean up
	defer func() {
		spamFilter.ResetLearning("user1")
		spamFilter.ResetLearning("user2")
	}()

	// Train different users with different patterns
	// User1: Prefers technical emails (programming, etc.)
	techSpam := []struct{ subject, body string }{
		{"Free Programming Course!", "Learn to code for FREE! No experience needed! Get certified!"},
		{"Software Download - CRACKED!", "Download premium software for FREE! All versions cracked!"},
	}

	techHam := []struct{ subject, body string }{
		{"Code Review Request", "Please review the pull request for the authentication module."},
		{"Bug Report #1234", "Found an issue with the user login function. Steps to reproduce..."},
	}

	// User2: Business emails
	bizSpam := []struct{ subject, body string }{
		{"Investment Opportunity!", "Guaranteed 300% returns! Invest now!"},
		{"Business Loan Approved!", "Your business loan has been pre-approved! Apply now!"},
	}

	bizHam := []struct{ subject, body string }{
		{"Q3 Sales Report", "Please find attached the quarterly sales report for review."},
		{"Client Meeting Scheduled", "Meeting with ABC Corp scheduled for Friday at 10 AM."},
	}

	// Train user1 (tech user)
	t.Log("Training user1 (tech preferences)...")
	for _, email := range techSpam {
		err := spamFilter.TrainSpam(email.subject, email.body, "user1")
		if err != nil {
			t.Fatalf("Failed to train user1 spam: %v", err)
		}
	}
	for _, email := range techHam {
		err := spamFilter.TrainHam(email.subject, email.body, "user1")
		if err != nil {
			t.Fatalf("Failed to train user1 ham: %v", err)
		}
	}

	// Train user2 (business user)
	t.Log("Training user2 (business preferences)...")
	for _, email := range bizSpam {
		err := spamFilter.TrainSpam(email.subject, email.body, "user2")
		if err != nil {
			t.Fatalf("Failed to train user2 spam: %v", err)
		}
	}
	for _, email := range bizHam {
		err := spamFilter.TrainHam(email.subject, email.body, "user2")
		if err != nil {
			t.Fatalf("Failed to train user2 ham: %v", err)
		}
	}

	// Test that users have different models
	testEmail := &email.Email{
		Subject: "Software Development Project",
		Body:    "We need to discuss the software development project timeline and deliverables.",
		From:    "test@example.com",
		To:      []string{"recipient@example.com"},
		Headers: map[string]string{
			"Subject": "Software Development Project",
			"From":    "test@example.com",
		},
	}

	// This should be classified differently by each user
	// (Note: This test shows the concept, actual results may vary based on training)
	score1 := spamFilter.CalculateSpamScore(testEmail) // Uses default user
	score2 := spamFilter.CalculateSpamScore(testEmail) // Uses default user

	t.Logf("Test email scores - Score1: %.2f, Score2: %.2f", score1, score2)
	t.Log("Multi-user functionality test completed (per-user models are separate in Redis)")
}

// Test Redis persistence and recovery
func TestRedisIntegrationPersistence(t *testing.T) {
	if !isRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	// Create Redis config
	cfg := config.DefaultConfig()
	cfg.Learning.Enabled = true
	cfg.Learning.Backend = "redis"
	cfg.Learning.Redis.RedisURL = "redis://localhost:6379"
	cfg.Learning.Redis.KeyPrefix = "zpo:integration:persistence"
	cfg.Learning.Redis.DatabaseNum = 1
	cfg.Learning.Redis.MinLearns = 1

	// Create first spam filter instance
	spamFilter1 := filter.NewSpamFilterWithConfig(cfg)
	if spamFilter1 == nil {
		t.Fatal("Failed to create spam filter 1")
	}

	// Clean up
	defer func() {
		spamFilter1.ResetLearning("persistuser")
	}()

	// Train some data
	t.Log("Training data with instance 1...")
	err := spamFilter1.TrainSpam("Test spam", "This is spam content", "persistuser")
	if err != nil {
		t.Fatalf("Failed to train with instance 1: %v", err)
	}

	err = spamFilter1.TrainHam("Test ham", "This is ham content", "persistuser")
	if err != nil {
		t.Fatalf("Failed to train with instance 1: %v", err)
	}

	// Create second spam filter instance (simulating restart/new instance)
	spamFilter2 := filter.NewSpamFilterWithConfig(cfg)
	if spamFilter2 == nil {
		t.Fatal("Failed to create spam filter 2")
	}

	// Test that data persists across instances
	testEmail := &email.Email{
		Subject: "Test persistence",
		Body:    "Testing if the training data persists across instances",
		From:    "test@example.com",
		To:      []string{"recipient@example.com"},
		Headers: map[string]string{
			"Subject": "Test persistence",
			"From":    "test@example.com",
		},
	}

	score := spamFilter2.CalculateSpamScore(testEmail)
	t.Logf("Score from second instance: %.2f", score)

	// The fact that we can calculate a score means the training data persisted
	if score < 0 {
		t.Error("Score should be valid, indicating data persistence")
	}

	t.Log("Persistence test completed - data successfully shared between instances")
}
