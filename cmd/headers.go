package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpam/spam-filter/pkg/config"
	"github.com/zpam/spam-filter/pkg/email"
	"github.com/zpam/spam-filter/pkg/headers"
)

// headersCmd represents the headers command
var headersCmd = &cobra.Command{
	Use:   "headers [email-file]",
	Short: "Validate email headers (SPF/DKIM/DMARC)",
	Long: `Analyze email headers for authentication validity and suspicious patterns.

This command validates:
- SPF (Sender Policy Framework) records
- DKIM (DomainKeys Identified Mail) signatures  
- DMARC (Domain-based Message Authentication, Reporting & Conformance) policies
- Email routing path analysis
- Header anomaly detection

Examples:
  zpam headers email.eml
  zpam headers email.eml --json
  zpam headers email.eml --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runHeaders,
}

var (
	headersJSON    bool
	headersVerbose bool
)

func init() {
	rootCmd.AddCommand(headersCmd)

	headersCmd.Flags().BoolVar(&headersJSON, "json", false, "Output results in JSON format")
	headersCmd.Flags().BoolVar(&headersVerbose, "verbose", false, "Show verbose validation details")
}

func runHeaders(cmd *cobra.Command, args []string) error {
	emailFile := args[0]

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	// Parse email
	parser := email.NewParser()
	emailObj, err := parser.ParseFromFile(emailFile)
	if err != nil {
		return fmt.Errorf("failed to parse email: %v", err)
	}

	// Create headers validator
	headersConfig := &headers.Config{
		EnableSPF:             cfg.Headers.EnableSPF,
		EnableDKIM:            cfg.Headers.EnableDKIM,
		EnableDMARC:           cfg.Headers.EnableDMARC,
		DNSTimeout:            time.Duration(cfg.Headers.DNSTimeoutMs) * time.Millisecond,
		MaxHopCount:           cfg.Headers.MaxHopCount,
		SuspiciousServerScore: cfg.Headers.SuspiciousServerScore,
		CacheSize:             cfg.Headers.CacheSize,
		CacheTTL:              time.Duration(cfg.Headers.CacheTTLMin) * time.Minute,
		SuspiciousServers: []string{
			"suspicious", "spam", "bulk", "mass", "marketing",
			"promo", "offer", "deal", "free", "win",
		},
		OpenRelayPatterns: []string{
			"unknown", "dynamic", "dhcp", "dial", "cable",
			"dsl", "adsl", "pool", "client", "user",
		},
	}

	validator := headers.NewValidator(headersConfig)

	// Validate headers
	result := validator.ValidateHeaders(emailObj.Headers)

	// Output results
	if headersJSON {
		return outputJSON(result)
	}

	return outputText(result, headersVerbose)
}

func outputJSON(result *headers.ValidationResult) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

func outputText(result *headers.ValidationResult, verbose bool) error {
	fmt.Printf("=== Email Headers Validation Results ===\n\n")

	// Overall scores
	fmt.Printf("📊 Overall Scores:\n")
	fmt.Printf("   Authentication Score: %.1f/100 %s\n", result.AuthScore, getScoreEmoji(result.AuthScore))
	fmt.Printf("   Suspicious Score:     %.1f/100 %s\n", result.SuspiciScore, getSuspiciousEmoji(result.SuspiciScore))
	fmt.Printf("   Validation Time:      %v\n\n", result.Duration)

	// SPF Results
	fmt.Printf("🔐 SPF (Sender Policy Framework):\n")
	fmt.Printf("   Status: %s %s\n", result.SPF.Result, getSPFEmoji(result.SPF.Result))
	if result.SPF.Record != "" {
		fmt.Printf("   Record: %s\n", result.SPF.Record)
	}
	if result.SPF.Explanation != "" {
		fmt.Printf("   Details: %s\n", result.SPF.Explanation)
	}
	if len(result.SPF.IPMatches) > 0 {
		fmt.Printf("   IP Matches: %s\n", strings.Join(result.SPF.IPMatches, ", "))
	}
	fmt.Println()

	// DKIM Results
	fmt.Printf("🔑 DKIM (DomainKeys Identified Mail):\n")
	fmt.Printf("   Valid: %s %s\n", formatBool(result.DKIM.Valid), getDKIMEmoji(result.DKIM.Valid))
	if len(result.DKIM.Domains) > 0 {
		fmt.Printf("   Domains: %s\n", strings.Join(result.DKIM.Domains, ", "))
	}
	if len(result.DKIM.Selectors) > 0 {
		fmt.Printf("   Selectors: %s\n", strings.Join(result.DKIM.Selectors, ", "))
	}
	if len(result.DKIM.Algorithms) > 0 {
		fmt.Printf("   Algorithms: %s\n", strings.Join(result.DKIM.Algorithms, ", "))
	}
	if result.DKIM.Explanation != "" {
		fmt.Printf("   Details: %s\n", result.DKIM.Explanation)
	}
	fmt.Println()

	// DMARC Results
	fmt.Printf("🛡️  DMARC (Domain-based Message Authentication):\n")
	fmt.Printf("   Valid: %s %s\n", formatBool(result.DMARC.Valid), getDMARCEmoji(result.DMARC.Valid))
	if result.DMARC.Policy != "" {
		fmt.Printf("   Policy: %s\n", result.DMARC.Policy)
	}
	if result.DMARC.Alignment != "" {
		fmt.Printf("   Alignment: %s\n", result.DMARC.Alignment)
	}
	if result.DMARC.Percentage > 0 {
		fmt.Printf("   Percentage: %d%%\n", result.DMARC.Percentage)
	}
	if result.DMARC.Explanation != "" {
		fmt.Printf("   Details: %s\n", result.DMARC.Explanation)
	}
	fmt.Println()

	// Routing Analysis
	fmt.Printf("🌐 Routing Analysis:\n")
	fmt.Printf("   Total Hops: %d\n", result.Routing.HopCount)

	if len(result.Routing.SuspiciousHops) > 0 {
		fmt.Printf("   ⚠️  Suspicious Hops:\n")
		for _, hop := range result.Routing.SuspiciousHops {
			fmt.Printf("      - %s\n", hop)
		}
	}

	if len(result.Routing.OpenRelays) > 0 {
		fmt.Printf("   🔓 Open Relays:\n")
		for _, relay := range result.Routing.OpenRelays {
			fmt.Printf("      - %s\n", relay)
		}
	}

	if len(result.Routing.ReverseDNSIssues) > 0 {
		fmt.Printf("   🔍 Reverse DNS Issues:\n")
		for _, issue := range result.Routing.ReverseDNSIssues {
			fmt.Printf("      - %s\n", issue)
		}
	}

	if len(result.Routing.GeoAnomalies) > 0 {
		fmt.Printf("   🌍 Geographic Anomalies:\n")
		for _, anomaly := range result.Routing.GeoAnomalies {
			fmt.Printf("      - %s\n", anomaly)
		}
	}

	if len(result.Routing.TimingAnomalies) > 0 {
		fmt.Printf("   ⏰ Timing Anomalies:\n")
		for _, anomaly := range result.Routing.TimingAnomalies {
			fmt.Printf("      - %s\n", anomaly)
		}
	}

	fmt.Println()

	// Header Anomalies
	if len(result.Anomalies) > 0 {
		fmt.Printf("❌ Header Anomalies:\n")
		for _, anomaly := range result.Anomalies {
			fmt.Printf("   - %s\n", anomaly)
		}
		fmt.Println()
	}

	// Verbose output
	if verbose {
		fmt.Printf("=== Detailed Analysis ===\n\n")

		// Add more detailed information
		fmt.Printf("SPF Record Details:\n")
		fmt.Printf("  Record: %s\n", result.SPF.Record)
		fmt.Printf("  Result: %s\n", result.SPF.Result)
		fmt.Printf("  Explanation: %s\n\n", result.SPF.Explanation)

		if len(result.DKIM.Signatures) > 0 {
			fmt.Printf("DKIM Signatures:\n")
			for i, sig := range result.DKIM.Signatures {
				fmt.Printf("  Signature %d: %s\n", i+1, sig)
			}
			fmt.Println()
		}

		fmt.Printf("Validation Performance:\n")
		fmt.Printf("  Started: %s\n", result.ValidatedAt.Format(time.RFC3339))
		fmt.Printf("  Duration: %v\n", result.Duration)
		fmt.Printf("  Rate: %.2f validations/sec\n", 1.0/result.Duration.Seconds())
	}

	// Final assessment
	fmt.Printf("=== Final Assessment ===\n")

	if result.AuthScore >= 80 && result.SuspiciScore <= 20 {
		fmt.Printf("✅ LEGITIMATE - Strong authentication, low suspicious activity\n")
	} else if result.AuthScore >= 60 && result.SuspiciScore <= 40 {
		fmt.Printf("⚠️  QUESTIONABLE - Moderate authentication, some suspicious indicators\n")
	} else if result.AuthScore >= 40 && result.SuspiciScore <= 60 {
		fmt.Printf("🔶 SUSPICIOUS - Weak authentication, notable suspicious activity\n")
	} else {
		fmt.Printf("🚨 HIGHLY SUSPICIOUS - Poor authentication, high suspicious activity\n")
	}

	return nil
}

// Helper functions for formatting

func getScoreEmoji(score float64) string {
	if score >= 80 {
		return "✅"
	} else if score >= 60 {
		return "⚠️"
	} else if score >= 40 {
		return "🔶"
	} else {
		return "❌"
	}
}

func getSuspiciousEmoji(score float64) string {
	if score <= 20 {
		return "✅"
	} else if score <= 40 {
		return "⚠️"
	} else if score <= 60 {
		return "🔶"
	} else {
		return "🚨"
	}
}

func getSPFEmoji(result string) string {
	switch result {
	case "pass":
		return "✅"
	case "fail":
		return "❌"
	case "softfail":
		return "⚠️"
	case "neutral":
		return "🔶"
	case "none":
		return "❓"
	default:
		return "❓"
	}
}

func getDKIMEmoji(valid bool) string {
	if valid {
		return "✅"
	}
	return "❌"
}

func getDMARCEmoji(valid bool) string {
	if valid {
		return "✅"
	}
	return "❌"
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
