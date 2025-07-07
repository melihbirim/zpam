package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

var generatePluginCmd = &cobra.Command{
	Use:   "plugin [name]",
	Short: "Generate a new plugin template",
	Long: `Generate a new plugin template with boilerplate code.

This command creates:
- Plugin source file (pkg/plugins/[name].go)
- Plugin test file (pkg/plugins/[name]_test.go)
- Configuration example
- Registration instructions

Example:
  zpo generate plugin my_awesome_plugin`,
	Args: cobra.ExactArgs(1),
	Run:  runGeneratePlugin,
}

var (
	pluginType    string
	pluginAuthor  string
	pluginPackage string
	outputDir     string
	overwrite     bool
)

// EmailGenerator generates realistic test emails
type EmailGenerator struct {
	rand *rand.Rand

	// Email templates and data
	spamSubjects []string
	hamSubjects  []string
	spamBodies   []string
	hamBodies    []string
	spamDomains  []string
	hamDomains   []string
	spamKeywords []string
	hamKeywords  []string
	names        []string
	companies    []string
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

	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generatePluginCmd)

	// Plugin generation flags
	generatePluginCmd.Flags().StringVarP(&pluginType, "type", "t", "content", "Plugin type (content, reputation, ml, external, rules)")
	generatePluginCmd.Flags().StringVarP(&pluginAuthor, "author", "a", "", "Plugin author name")
	generatePluginCmd.Flags().StringVarP(&pluginPackage, "package", "p", "plugins", "Go package name")
	generatePluginCmd.Flags().StringVarP(&outputDir, "output", "o", "pkg/plugins", "Output directory")
	generatePluginCmd.Flags().BoolVarP(&overwrite, "overwrite", "f", false, "Overwrite existing files")
}

type PluginTemplate struct {
	Name            string
	PackageName     string
	StructName      string
	FunctionName    string
	Author          string
	Description     string
	Interface       string
	InterfaceMethod string
	ExampleConfig   string
}

func runGeneratePlugin(cmd *cobra.Command, args []string) {
	pluginName := args[0]

	// Validate plugin name
	if !isValidPluginName(pluginName) {
		fmt.Printf("Error: Plugin name '%s' is not valid. Use lowercase letters, numbers, and underscores only.\n", pluginName)
		os.Exit(1)
	}

	// Prepare template data
	tmpl := &PluginTemplate{
		Name:         pluginName,
		PackageName:  pluginPackage,
		StructName:   toCamelCase(pluginName) + "Plugin",
		FunctionName: "New" + toCamelCase(pluginName) + "Plugin",
		Author:       pluginAuthor,
		Description:  fmt.Sprintf("%s plugin for ZPO spam filter", strings.Title(strings.ReplaceAll(pluginName, "_", " "))),
	}

	// Set interface based on type
	switch pluginType {
	case "content":
		tmpl.Interface = "ContentAnalyzer"
		tmpl.InterfaceMethod = "AnalyzeContent"
	case "reputation":
		tmpl.Interface = "ReputationChecker"
		tmpl.InterfaceMethod = "CheckReputation"
	case "ml":
		tmpl.Interface = "MLClassifier"
		tmpl.InterfaceMethod = "Classify"
	case "external":
		tmpl.Interface = "ExternalEngine"
		tmpl.InterfaceMethod = "Analyze"
	case "rules":
		tmpl.Interface = "CustomRuleEngine"
		tmpl.InterfaceMethod = "EvaluateRules"
	default:
		fmt.Printf("Error: Unknown plugin type '%s'. Use: content, reputation, ml, external, rules\n", pluginType)
		os.Exit(1)
	}

	// Generate files
	if err := generatePluginFiles(tmpl); err != nil {
		fmt.Printf("Error generating plugin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Plugin '%s' generated successfully!\n\n", pluginName)
	fmt.Println("Next steps:")
	fmt.Printf("1. Edit the generated files in %s/\n", outputDir)
	fmt.Printf("2. Implement the %s method\n", tmpl.InterfaceMethod)
	fmt.Printf("3. Add plugin registration to pkg/filter/spam_filter.go:\n")
	fmt.Printf("   sf.pluginManager.RegisterPlugin(plugins.%s())\n", tmpl.FunctionName)
	fmt.Printf("4. Add plugin to cmd/plugins.go CLI commands\n")
	fmt.Printf("5. Add configuration to config.yaml\n")
	fmt.Printf("6. Test your plugin:\n")
	fmt.Printf("   ./zpo plugins test-one %s examples/test_headers.eml\n", pluginName)
}

func generatePluginFiles(tmpl *PluginTemplate) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate main plugin file
	if err := generatePluginSource(tmpl); err != nil {
		return fmt.Errorf("failed to generate plugin source: %w", err)
	}

	// Generate test file
	if err := generatePluginTest(tmpl); err != nil {
		return fmt.Errorf("failed to generate plugin test: %w", err)
	}

	// Generate README
	if err := generatePluginReadme(tmpl); err != nil {
		return fmt.Errorf("failed to generate plugin README: %w", err)
	}

	return nil
}

func generatePluginSource(tmpl *PluginTemplate) error {
	filename := filepath.Join(outputDir, tmpl.Name+".go")

	if !overwrite && fileExists(filename) {
		return fmt.Errorf("file %s already exists (use --overwrite to overwrite)", filename)
	}

	sourceTemplate := `// Package {{.PackageName}} - {{.Description}}
{{if .Author}}// Author: {{.Author}}{{end}}
// Generated by ZPO plugin generator

package {{.PackageName}}

import (
	"context"
	"fmt"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

// {{.StructName}} implements the {{.Interface}} interface
type {{.StructName}} struct {
	config  *PluginConfig
	enabled bool
	stats   *{{.StructName}}Stats
	
	// TODO: Add your custom fields here
	// apiKey     string
	// endpoint   string
	// timeout    time.Duration
	// maxRetries int
}

// {{.StructName}}Stats tracks plugin performance
type {{.StructName}}Stats struct {
	RequestsTotal   int64   ` + "`json:\"requests_total\"`" + `
	RequestsFailed  int64   ` + "`json:\"requests_failed\"`" + `
	AverageResponse float64 ` + "`json:\"average_response_ms\"`" + `
	LastUpdate      string  ` + "`json:\"last_update\"`" + `
}

// {{.FunctionName}} creates a new instance of {{.StructName}}
func {{.FunctionName}}() *{{.StructName}} {
	return &{{.StructName}}{
		stats: &{{.StructName}}Stats{},
		// TODO: Initialize default values
		// timeout:    10 * time.Second,
		// maxRetries: 3,
	}
}

// Name returns the plugin name
func (p *{{.StructName}}) Name() string {
	return "{{.Name}}"
}

// Version returns the plugin version
func (p *{{.StructName}}) Version() string {
	return "1.0.0"
}

// Description returns a brief description of what the plugin does
func (p *{{.StructName}}) Description() string {
	return "{{.Description}}"
}

// Initialize sets up the plugin with configuration
func (p *{{.StructName}}) Initialize(config *PluginConfig) error {
	p.config = config
	p.enabled = config.Enabled

	if !p.enabled {
		return nil
	}

	// TODO: Extract configuration settings
	if config.Settings != nil {
		// Example configuration extraction:
		// if apiKey, ok := config.Settings["api_key"].(string); ok {
		//     p.apiKey = apiKey
		// }
		// 
		// if endpoint, ok := config.Settings["endpoint"].(string); ok {
		//     p.endpoint = endpoint
		// } else {
		//     p.endpoint = "https://api.example.com/v1/check" // Default
		// }
	}

	// TODO: Validate required configuration
	// if p.apiKey == "" {
	//     return fmt.Errorf("api_key is required for {{.Name}} plugin")
	// }

	// TODO: Initialize any connections, load models, etc.
	if err := p.initializeResources(); err != nil {
		return fmt.Errorf("failed to initialize {{.Name}} plugin: %v", err)
	}

	return nil
}

// initializeResources sets up any resources needed by the plugin
func (p *{{.StructName}}) initializeResources() error {
	// TODO: Initialize databases, API clients, ML models, etc.
	// Example:
	// p.apiClient = &http.Client{Timeout: p.timeout}
	
	return nil
}

// IsHealthy checks if the plugin is ready to process emails
func (p *{{.StructName}}) IsHealthy(ctx context.Context) error {
	if !p.enabled {
		return fmt.Errorf("plugin not enabled")
	}

	// TODO: Implement health checks
	// Examples:
	// - Ping external API
	// - Check database connection
	// - Verify model files exist
	// - Test authentication
	
	return nil
}

// Cleanup releases any resources when the plugin is shutting down
func (p *{{.StructName}}) Cleanup() error {
	// TODO: Close connections, save state, etc.
	// Examples:
	// - Close database connections
	// - Save ML model state
	// - Clean up temporary files
	
	return nil
}

// {{.InterfaceMethod}} implements the {{.Interface}} interface
func (p *{{.StructName}}) {{.InterfaceMethod}}(ctx context.Context, email *email.Email) (*PluginResult, error) {
	start := time.Now()
	
	result := &PluginResult{
		Name:        p.Name(),
		Score:       0,
		Confidence:  0.5,
		Details:     make(map[string]any),
		Metadata:    make(map[string]string),
		Rules:       []string{},
		ProcessTime: 0,
	}

	if !p.enabled {
		result.Error = fmt.Errorf("plugin not enabled")
		result.ProcessTime = time.Since(start)
		return result, nil
	}

	// TODO: Implement your {{.Interface}} logic here
	// Examples based on plugin type:
	
	{{if eq .Interface "ContentAnalyzer"}}// Content analysis example:
	// - Check subject line for spam keywords
	// - Analyze body text patterns
	// - Check HTML structure
	// - Evaluate text statistics
	
	// Example implementation:
	// score, confidence, rules := p.analyzeContent(email)
	// result.Score = score
	// result.Confidence = confidence
	// result.Rules = rules{{end}}
	
	{{if eq .Interface "ReputationChecker"}}// Reputation checking example:
	// - Check sender domain reputation
	// - Verify IP address reputation
	// - Look up URL reputation
	// - Check against blacklists
	
	// Example implementation:
	// domain := p.extractDomain(email.From)
	// reputation, err := p.checkDomainReputation(ctx, domain)
	// if err != nil {
	//     return result, err
	// }
	// result.Score = reputation.Score
	// result.Confidence = reputation.Confidence{{end}}
	
	{{if eq .Interface "MLClassifier"}}// ML classification example:
	// - Extract features from email
	// - Run ML model inference
	// - Interpret prediction results
	
	// Example implementation:
	// features := p.extractFeatures(email)
	// prediction, err := p.classify(features)
	// if err != nil {
	//     return result, err
	// }
	// result.Score = prediction.SpamScore * 100
	// result.Confidence = prediction.Confidence
	// if prediction.IsSpam {
	//     result.Rules = append(result.Rules, "ML classified as spam")
	// }{{end}}
	
	{{if eq .Interface "ExternalEngine"}}// External service example:
	// - Prepare API request
	// - Call external service
	// - Parse response
	// - Handle errors and retries
	
	// Example implementation:
	// response, err := p.callExternalAPI(ctx, email)
	// if err != nil {
	//     return result, err
	// }
	// result.Score = response.Score
	// result.Confidence = response.Confidence
	// result.Rules = response.Rules{{end}}
	
	{{if eq .Interface "CustomRuleEngine"}}// Custom rules example:
	// - Load rule definitions
	// - Evaluate each rule against email
	// - Calculate combined score
	
	// Example implementation:
	// triggeredRules := p.evaluateRules(email)
	// totalScore := 0.0
	// for _, rule := range triggeredRules {
	//     totalScore += rule.Score
	//     result.Rules = append(result.Rules, rule.Description)
	// }
	// result.Score = totalScore{{end}}

	// Add metadata
	result.Metadata["plugin_version"] = p.Version()
	result.Metadata["analysis_time"] = fmt.Sprintf("%.2fms", time.Since(start).Seconds()*1000)
	
	// Update statistics
	p.updateStats(time.Since(start), nil)
	
	result.ProcessTime = time.Since(start)
	return result, nil
}

// TODO: Add your helper methods here
// Examples:

// extractDomain extracts domain from email address
func (p *{{.StructName}}) extractDomain(emailAddr string) string {
	parts := strings.Split(emailAddr, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return emailAddr
}

// updateStats updates plugin performance statistics
func (p *{{.StructName}}) updateStats(duration time.Duration, err error) {
	p.stats.RequestsTotal++
	if err != nil {
		p.stats.RequestsFailed++
	}
	
	// Update average response time
	responseTime := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
	if p.stats.RequestsTotal == 1 {
		p.stats.AverageResponse = responseTime
	} else {
		p.stats.AverageResponse = (p.stats.AverageResponse*float64(p.stats.RequestsTotal-1) + responseTime) / float64(p.stats.RequestsTotal)
	}
	
	p.stats.LastUpdate = time.Now().Format(time.RFC3339)
}

// GetStats returns plugin statistics (optional)
func (p *{{.StructName}}) GetStats() *{{.StructName}}Stats {
	return p.stats
}
`

	return executeTemplate(sourceTemplate, tmpl, filename)
}

func generatePluginTest(tmpl *PluginTemplate) error {
	filename := filepath.Join(outputDir, tmpl.Name+"_test.go")

	if !overwrite && fileExists(filename) {
		return nil // Skip test file if exists and not overwriting
	}

	testTemplate := `package {{.PackageName}}

import (
	"context"
	"testing"
	"time"

	"github.com/zpo/spam-filter/pkg/email"
)

func Test{{.StructName}}_{{.InterfaceMethod}}(t *testing.T) {
	plugin := {{.FunctionName}}()
	
	config := &PluginConfig{
		Enabled: true,
		Settings: map[string]any{
			// TODO: Add test configuration
			// "api_key": "test-key",
		},
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}
	
	testEmail := &email.Email{
		Subject: "Test Subject",
		Body:    "Test body content",
		From:    "test@example.com",
		Headers: map[string]string{
			"Message-ID": "test@example.com",
			"Date":       time.Now().Format(time.RFC822),
		},
	}
	
	ctx := context.Background()
	result, err := plugin.{{.InterfaceMethod}}(ctx, testEmail)
	
	if err != nil {
		t.Fatalf("{{.InterfaceMethod}} failed: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result.Name != plugin.Name() {
		t.Errorf("Expected name %s, got %s", plugin.Name(), result.Name)
	}
	
	// TODO: Add more specific tests based on your plugin logic
	// Examples:
	// - Test specific spam indicators
	// - Verify score calculations
	// - Check rule triggering
	// - Validate metadata
}

func Test{{.StructName}}_Configuration(t *testing.T) {
	plugin := {{.FunctionName}}()
	
	// Test with missing required config
	config := &PluginConfig{
		Enabled:  true,
		Settings: map[string]any{},
	}
	
	err := plugin.Initialize(config)
	// TODO: Update this test based on your configuration requirements
	// if err == nil {
	//     t.Error("Expected error for missing required config, got nil")
	// }
	
	_ = err // Remove this line when implementing the test
}

func Test{{.StructName}}_HealthCheck(t *testing.T) {
	plugin := {{.FunctionName}}()
	
	config := &PluginConfig{
		Enabled: true,
		Settings: map[string]any{
			// TODO: Add valid test configuration
		},
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}
	
	ctx := context.Background()
	err = plugin.IsHealthy(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func Benchmark{{.StructName}}_{{.InterfaceMethod}}(b *testing.B) {
	plugin := {{.FunctionName}}()
	config := &PluginConfig{
		Enabled: true,
		Settings: map[string]any{
			// TODO: Add benchmark configuration
		},
	}
	
	err := plugin.Initialize(config)
	if err != nil {
		b.Fatalf("Failed to initialize plugin: %v", err)
	}
	
	testEmail := &email.Email{
		Subject: "Benchmark Test Subject",
		Body:    "Benchmark test body content with some example text",
		From:    "benchmark@example.com",
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.{{.InterfaceMethod}}(ctx, testEmail)
		if err != nil {
			b.Fatal(err)
		}
	}
}
`

	return executeTemplate(testTemplate, tmpl, filename)
}

func generatePluginReadme(tmpl *PluginTemplate) error {
	filename := filepath.Join(outputDir, tmpl.Name+"_README.md")

	if !overwrite && fileExists(filename) {
		return nil // Skip if exists and not overwriting
	}

	readmeTemplate := `# {{.StructName}}

{{.Description}}

{{if .Author}}**Author:** {{.Author}}{{end}}
**Version:** 1.0.0  
**Type:** {{.Interface}}

## Description

TODO: Describe what your plugin does, how it works, and what spam indicators it detects.

## Configuration

Add this to your ` + "`config.yaml`" + `:

` + "```yaml" + `
plugins:
  {{.Name}}:
    enabled: true
    weight: 2.0
    priority: 10
    timeout_ms: 5000
    settings:
      # TODO: Add your configuration options here
      # api_key: "your-api-key"
      # endpoint: "https://api.example.com/v1/check"
      # timeout: 10000
      # max_retries: 3
      # custom_setting: "value"
` + "```" + `

### Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| TODO | string | yes | - | Description |

## Features

TODO: List the key features of your plugin:

- Feature 1
- Feature 2
- Feature 3

## Usage

### Testing

Test your plugin with a specific email:

` + "```bash" + `
./zpo plugins test-one {{.Name}} examples/test_headers.eml
` + "```" + `

Test with all plugins enabled:

` + "```bash" + `
./zpo plugins test examples/test_headers.eml
` + "```" + `

### Production

Enable the plugin in your production configuration and restart ZPO.

## Development

### Building

` + "```bash" + `
go build -o zpo .
` + "```" + `

### Testing

Run unit tests:

` + "```bash" + `
go test ./pkg/plugins/{{.Name}}_test.go
` + "```" + `

Run benchmarks:

` + "```bash" + `
go test -bench=. ./pkg/plugins/{{.Name}}_test.go
` + "```" + `

## Performance

TODO: Document expected performance characteristics:

- Processing time: < 100ms per email
- Memory usage: < 10MB
- External API calls: 1 per email (with caching)

## Troubleshooting

### Common Issues

1. **Issue 1**: Description and solution
2. **Issue 2**: Description and solution

### Debug Mode

Enable debug logging for this plugin:

` + "```yaml" + `
logging:
  level: debug
  plugins:
    {{.Name}}: debug
` + "```" + `

## License

TODO: Add your license information.
`

	return executeTemplate(readmeTemplate, tmpl, filename)
}

func executeTemplate(templateStr string, data *PluginTemplate, filename string) error {
	tmpl, err := template.New("plugin").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("Generated: %s\n", filename)
	return nil
}

func isValidPluginName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}

	return true
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
