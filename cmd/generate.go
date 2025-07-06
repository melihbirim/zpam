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
	
	// Create a long spam email with multiple paragraphs
	baseBody := fmt.Sprintf(template, link)
	
	// Add additional spam content to reach 1000+ words
	extraContent := []string{
		"This is a once-in-a-lifetime opportunity that you cannot afford to miss! Our revolutionary system has helped thousands of people achieve financial freedom and live the life of their dreams.",
		"Don't let this chance slip away - act now before it's too late! Our limited-time offer expires soon and you don't want to be left behind while others are making huge profits.",
		"We guarantee 100% satisfaction or your money back. No questions asked. This is completely risk-free and you have nothing to lose but everything to gain.",
		"Join our exclusive community of successful entrepreneurs who are already earning massive passive income. You could be next!",
		"URGENT: This offer is only available to the first 100 people who respond today. Don't wait - secure your spot now!",
		"Make money while you sleep! Our automated system works 24/7 to generate income for you. No experience required - anyone can do this!",
		"Stop struggling with financial problems. Take control of your future and start earning the money you deserve. Your dreams are within reach!",
		"WARNING: If you don't act now, you will regret it for the rest of your life. This opportunity may never come again.",
		"Testimonials from our satisfied customers: 'I made $50,000 in my first month!' 'This changed my life completely!' 'I wish I had found this sooner!'",
		"Free bonus included: Get our exclusive guide to making money online absolutely free when you sign up today. Limited quantities available!",
	}
	
	// Add 8-12 additional paragraphs to reach 1000+ words
	numParagraphs := 8 + g.rand.Intn(5)
	for i := 0; i < numParagraphs; i++ {
		paragraph := g.randomChoice(extraContent)
		// Add some randomization to avoid exact duplicates
		if g.rand.Float64() < 0.3 {
			paragraph = strings.ToUpper(paragraph)
		}
		if g.rand.Float64() < 0.5 {
			paragraph += " " + g.randomChoice(g.spamKeywords) + "! " + g.randomChoice(g.spamKeywords) + "!"
		}
		baseBody += "\n\n" + paragraph
	}
	
	// Add final call to action
	baseBody += "\n\nClick here immediately: " + link
	baseBody += "\n\nDon't wait! Act now! Time is running out!"
	
	return baseBody
}

// generateHamBody creates legitimate email body
func (g *EmailGenerator) generateHamBody() string {
	template := g.randomChoice(g.hamBodies)
	name := g.randomChoice(g.names)
	
	// Create a long legitimate email with detailed business content
	baseBody := fmt.Sprintf(template, name, name)
	
	// Add professional business content to reach 1000+ words
	businessContent := []string{
		"I wanted to provide you with a comprehensive update on our current project status and upcoming initiatives. We've made significant progress across all fronts.",
		"The quarterly analysis shows positive trends in key performance indicators. Revenue is up 15% compared to the same period last year, and customer satisfaction ratings have improved substantially.",
		"Our development team has been working diligently on the new features requested by stakeholders. The user interface improvements are scheduled for deployment next month.",
		"We've identified several opportunities for process optimization that could reduce operational costs by approximately 20% while maintaining service quality standards.",
		"The market research indicates strong demand for our upcoming product line. Initial feedback from focus groups has been overwhelmingly positive.",
		"Our partnership with regional distributors has expanded our market reach significantly. We're now serving customers in 15 additional metropolitan areas.",
		"The training program for new employees has been updated to include the latest industry best practices and compliance requirements. Initial results show improved retention rates.",
		"Budget allocations for the next fiscal year need to be finalized by the end of this month. Please review the preliminary numbers and provide your feedback.",
		"The client presentation scheduled for next week will showcase our innovative solutions and demonstrate our competitive advantages in the marketplace.",
		"Regulatory compliance updates require immediate attention. The new standards go into effect at the beginning of next quarter.",
		"Our sustainability initiatives have reduced energy consumption by 25% and waste output by 30%. These improvements align with our corporate responsibility goals.",
		"Customer feedback surveys indicate high satisfaction with recent service improvements. The average rating has increased from 4.2 to 4.7 out of 5.",
	}
	
	// Add 6-10 additional paragraphs for professional length
	numParagraphs := 6 + g.rand.Intn(5)
	for i := 0; i < numParagraphs; i++ {
		paragraph := g.randomChoice(businessContent)
		// Add some business-specific details
		if g.rand.Float64() < 0.4 {
			metrics := []string{"KPIs", "ROI", "quarterly targets", "market share", "customer acquisition"}
			paragraph += " The " + g.randomChoice(metrics) + " analysis supports this initiative."
		}
		baseBody += "\n\n" + paragraph
	}
	
	// Add professional closing
	baseBody += "\n\nPlease let me know if you have any questions or need additional information."
	baseBody += "\n\nI look forward to discussing this further in our upcoming meeting."
	baseBody += "\n\nThank you for your continued support and collaboration."
	
	return baseBody
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