package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	generateCount  int
	generateOutput string
	generateSplit  float64
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate test email data",
	Long:  `Generate diverse test email dataset for benchmarking and testing`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if generateCount <= 0 {
			return fmt.Errorf("count must be greater than 0")
		}

		if generateSplit < 0 || generateSplit > 1 {
			return fmt.Errorf("spam-ratio must be between 0 and 1")
		}

		generator := NewEmailGenerator()
		
		// Create output directory
		if err := os.MkdirAll(generateOutput, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
		
		// Calculate spam vs ham counts
		spamCount := int(float64(generateCount) * generateSplit)
		hamCount := generateCount - spamCount
		
		fmt.Printf("🧪 Generating test emails...\n")
		fmt.Printf("📧 Total emails: %d\n", generateCount)
		fmt.Printf("🚫 Spam emails: %d (%.1f%%)\n", spamCount, generateSplit*100)
		fmt.Printf("✅ Ham emails: %d (%.1f%%)\n", hamCount, (1-generateSplit)*100)
		fmt.Printf("📂 Output directory: %s\n\n", generateOutput)
		
		start := time.Now()
		
		// Generate spam emails
		for i := 0; i < spamCount; i++ {
			email := generator.GenerateSpamEmail()
			filename := filepath.Join(generateOutput, fmt.Sprintf("spam_%04d.eml", i+1))
			if err := os.WriteFile(filename, []byte(email), 0644); err != nil {
				return fmt.Errorf("failed to write spam email %d: %v", i+1, err)
			}
		}
		
		// Generate ham emails
		for i := 0; i < hamCount; i++ {
			email := generator.GenerateHamEmail()
			filename := filepath.Join(generateOutput, fmt.Sprintf("ham_%04d.eml", i+1))
			if err := os.WriteFile(filename, []byte(email), 0644); err != nil {
				return fmt.Errorf("failed to write ham email %d: %v", i+1, err)
			}
		}
		
		duration := time.Since(start)
		
		fmt.Printf("✅ Generation complete!\n")
		fmt.Printf("⏱️ Time taken: %v\n", duration)
		fmt.Printf("📈 Rate: %.0f emails/second\n", float64(generateCount)/duration.Seconds())
		
		return nil
	},
}

// EmailGenerator generates realistic test emails
type EmailGenerator struct {
	rand *rand.Rand
	
	// Email templates and data
	spamSubjects    []string
	hamSubjects     []string
	spamBodies      []string
	hamBodies       []string
	spamDomains     []string
	hamDomains      []string
	spamKeywords    []string
	hamKeywords     []string
	names           []string
	companies       []string
}

// NewEmailGenerator creates a new email generator
func NewEmailGenerator() *EmailGenerator {
	return &EmailGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
		
		spamSubjects: []string{
			"URGENT!!! FREE MONEY!!!",
			"You have won $1,000,000!!!",
			"ACT NOW - Limited time offer!",
			"Get rich quick - GUARANTEED!",
			"Nigerian Prince needs your help",
			"FREE Viagra - No prescription needed",
			"Lose 50 pounds in 10 days!",
			"Work from home - Make $5000/week",
			"CONGRATULATIONS - You're our winner!",
			"Click here for FREE gift cards",
			"Urgent: Your account will be closed",
			"Amazing investment opportunity",
		},
		
		hamSubjects: []string{
			"Meeting tomorrow at 2 PM",
			"Quarterly report attached",
			"Project update - Phase 2 complete",
			"Happy birthday!",
			"Weekend plans?",
			"Conference call notes",
			"Invoice #12345",
			"Welcome to our team",
			"System maintenance notice",
			"Monthly newsletter",
			"Re: Budget approval",
			"Lunch invitation",
		},
		
		spamBodies: []string{
			"Congratulations! You have been selected to receive FREE MONEY! No risk involved! GUARANTEED income! Act now before this offer expires! Click here: %s",
			"URGENT! Your account will be suspended unless you verify your information immediately! Click here to avoid suspension: %s",
			"Make money fast with our proven system! Thousands are already earning $10,000 per week! Join now: %s",
			"You have won our lottery! Claim your $1,000,000 prize now! Send your bank details to claim: %s",
			"Lose weight fast with our miracle pill! No diet or exercise needed! Order now: %s",
			"Get Viagra without prescription! Best prices guaranteed! Free shipping worldwide! Order: %s",
		},
		
		hamBodies: []string{
			"Hi there,\n\nI hope this email finds you well. I wanted to remind you about our meeting tomorrow at 2 PM in the conference room.\n\nWe'll be discussing the quarterly reports and planning for next quarter.\n\nPlease let me know if you need to reschedule.\n\nBest regards,\n%s",
			"Hello,\n\nPlease find attached the quarterly report for your review. The numbers look good overall, with a 15%% increase in revenue.\n\nLet me know if you have any questions.\n\nThanks,\n%s",
			"Hi team,\n\nJust a quick update on the project progress. Phase 2 has been completed successfully and we're on track for the deadline.\n\nNext steps:\n- Review deliverables\n- Prepare for Phase 3\n- Schedule team meeting\n\nBest,\n%s",
			"Dear %s,\n\nWe're planning a team lunch this Friday at 12:30 PM. Please let me know if you can make it.\n\nLooking forward to seeing everyone!\n\nRegards,\n%s",
		},
		
		spamDomains: []string{
			"get-rich-quick.com", "suspicious-domain.org", "free-money.net", "scam-alert.biz",
			"fake-bank.com", "phishing-site.net", "malware-host.org", "spam-central.com",
			"dodgy-pharma.net", "lottery-scam.org", "virus-download.com", "identity-theft.biz",
		},
		
		hamDomains: []string{
			"gmail.com", "yahoo.com", "outlook.com", "company.com", "university.edu",
			"government.gov", "nonprofit.org", "corporation.net", "startup.io", "tech-firm.com",
			"consulting.biz", "healthcare.org", "finance.com", "retail.net", "manufacturing.com",
		},
		
		spamKeywords: []string{
			"free money", "get rich", "make money fast", "guaranteed income", "no risk",
			"act now", "limited time", "urgent", "congratulations", "you have won",
			"lottery", "viagra", "lose weight", "work from home", "click here",
		},
		
		hamKeywords: []string{
			"meeting", "report", "project", "team", "schedule", "deadline", "budget",
			"invoice", "proposal", "update", "review", "conference", "training",
		},
		
		names: []string{
			"John Smith", "Jane Doe", "Mike Johnson", "Sarah Wilson", "David Brown",
			"Lisa Garcia", "Robert Miller", "Emily Davis", "Michael Anderson", "Jennifer Taylor",
			"Christopher Martinez", "Amanda Thomas", "Matthew Jackson", "Jessica White", "Daniel Harris",
		},
		
		companies: []string{
			"Tech Solutions Inc", "Global Dynamics", "Innovation Labs", "Future Systems",
			"Digital Ventures", "Smart Technologies", "Advanced Analytics", "Modern Enterprises",
			"NextGen Solutions", "Quantum Computing Co", "AI Innovations", "Cloud Services Ltd",
		},
	}
}

// GenerateSpamEmail generates a realistic spam email
func (g *EmailGenerator) GenerateSpamEmail() string {
	from := g.generateSpamSender()
	to := g.generateRecipient()
	subject := g.randomChoice(g.spamSubjects)
	body := g.generateSpamBody()
	
	// Add random spam characteristics
	subject = g.addSpamCharacteristics(subject)
	
	return g.formatEmail(from, to, subject, body)
}

// GenerateHamEmail generates a realistic ham email
func (g *EmailGenerator) GenerateHamEmail() string {
	from := g.generateHamSender()
	to := g.generateRecipient()
	subject := g.randomChoice(g.hamSubjects)
	body := g.generateHamBody()
	
	return g.formatEmail(from, to, subject, body)
}

// generateSpamSender creates a suspicious sender address
func (g *EmailGenerator) generateSpamSender() string {
	domain := g.randomChoice(g.spamDomains)
	usernames := []string{"noreply", "admin", "support", "winner", "lottery", "offer", "deals"}
	username := g.randomChoice(usernames)
	return fmt.Sprintf("%s@%s", username, domain)
}

// generateHamSender creates a legitimate sender address
func (g *EmailGenerator) generateHamSender() string {
	domain := g.randomChoice(g.hamDomains)
	name := g.randomChoice(g.names)
	nameParts := strings.Split(strings.ToLower(name), " ")
	username := fmt.Sprintf("%s.%s", nameParts[0], nameParts[1])
	return fmt.Sprintf("%s@%s", username, domain)
}

// generateRecipient creates a recipient address
func (g *EmailGenerator) generateRecipient() string {
	domains := []string{"example.com", "test.org", "demo.net", "sample.biz"}
	domain := g.randomChoice(domains)
	usernames := []string{"user", "customer", "employee", "member", "subscriber"}
	username := g.randomChoice(usernames)
	return fmt.Sprintf("%s@%s", username, domain)
}

// generateSpamBody creates spam email body
func (g *EmailGenerator) generateSpamBody() string {
	template := g.randomChoice(g.spamBodies)
	link := fmt.Sprintf("http://%s/click-here", g.randomChoice(g.spamDomains))
	return fmt.Sprintf(template, link)
}

// generateHamBody creates legitimate email body
func (g *EmailGenerator) generateHamBody() string {
	template := g.randomChoice(g.hamBodies)
	name := g.randomChoice(g.names)
	return fmt.Sprintf(template, name, name)
}

// addSpamCharacteristics adds typical spam characteristics
func (g *EmailGenerator) addSpamCharacteristics(subject string) string {
	// Randomly add excessive punctuation
	if g.rand.Float64() < 0.7 {
		subject = strings.ReplaceAll(subject, "!", "!!!")
	}
	
	// Randomly convert to uppercase
	if g.rand.Float64() < 0.5 {
		subject = strings.ToUpper(subject)
	}
	
	// Add extra spam keywords
	if g.rand.Float64() < 0.3 {
		keyword := g.randomChoice(g.spamKeywords)
		subject = fmt.Sprintf("%s - %s", subject, strings.ToUpper(keyword))
	}
	
	return subject
}

// formatEmail formats the email as EML
func (g *EmailGenerator) formatEmail(from, to, subject, body string) string {
	timestamp := time.Now().Add(-time.Duration(g.rand.Intn(365*24)) * time.Hour)
	
	return fmt.Sprintf(`From: %s
To: %s
Subject: %s
Date: %s
Message-ID: <%d@%s>

%s`,
		from,
		to,
		subject,
		timestamp.Format("Mon, 02 Jan 2006 15:04:05 -0700"),
		g.rand.Int63(),
		"generator.local",
		body,
	)
}

// randomChoice selects a random item from slice
func (g *EmailGenerator) randomChoice(items []string) string {
	return items[g.rand.Intn(len(items))]
}

func init() {
	generateCmd.Flags().IntVarP(&generateCount, "count", "n", 100, "Number of emails to generate")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "test-data", "Output directory")
	generateCmd.Flags().Float64VarP(&generateSplit, "spam-ratio", "r", 0.3, "Ratio of spam emails (0.0-1.0)")
} 