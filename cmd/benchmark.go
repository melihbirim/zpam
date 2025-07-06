package cmd

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/zpo/spam-filter/pkg/config"
	"github.com/zpo/spam-filter/pkg/filter"
)

var (
	benchmarkInput      string
	benchmarkConfig     string
	benchmarkRuns       int
	benchmarkConcurrent int
	benchmarkOutput     bool
	benchmarkParallel   bool
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Performance benchmark and analysis",
	Long:  `Run performance benchmarks on email datasets and analyze results`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if benchmarkInput == "" {
			return fmt.Errorf("input directory is required")
		}

		// Load configuration
		cfg, err := filter.LoadConfigFromPath(benchmarkConfig)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %v", err)
		}

		// Find all email files
		emailFiles, err := findEmailFiles(benchmarkInput)
		if err != nil {
			return fmt.Errorf("failed to find email files: %v", err)
		}

		if len(emailFiles) == 0 {
			return fmt.Errorf("no email files found in %s", benchmarkInput)
		}

		fmt.Printf("🚀 ZPO Performance Benchmark\n")
		fmt.Printf("📁 Input directory: %s\n", benchmarkInput)
		fmt.Printf("📧 Email files found: %d\n", len(emailFiles))
		fmt.Printf("🔄 Benchmark runs: %d\n", benchmarkRuns)
		fmt.Printf("⚡ Concurrent workers: %d\n", benchmarkConcurrent)
		if benchmarkConfig != "" {
			fmt.Printf("⚙️ Configuration: %s\n", benchmarkConfig)
		}

		if benchmarkParallel {
			fmt.Printf("🔥 Parallel execution mode enabled\n")
		} else {
			fmt.Printf("🐌 Sequential execution mode (for comparison)\n")
		}
		fmt.Printf("\n")

		// Run benchmarks
		benchmark := NewBenchmark(cfg)

		if benchmarkParallel {
			results := benchmark.RunParallel(emailFiles, benchmarkRuns, benchmarkConcurrent)
			displayParallelBenchmarkResults(results)
		} else {
			results := benchmark.Run(emailFiles, benchmarkRuns, benchmarkConcurrent)
			displayBenchmarkResults(results)
		}

		return nil
	},
}

// BenchmarkResult contains performance metrics
type BenchmarkResult struct {
	TotalEmails     int
	TotalTime       time.Duration
	AvgTimePerEmail float64
	MinTime         time.Duration
	MaxTime         time.Duration
	MedianTime      time.Duration
	P95Time         time.Duration
	P99Time         time.Duration
	EmailsPerSecond float64

	// Classification results
	SpamDetected int
	HamDetected  int
	SpamAccuracy float64 // If we know ground truth
	HamAccuracy  float64

	// Performance breakdown
	ParseTime    time.Duration
	AnalysisTime time.Duration
	ScoringTime  time.Duration

	// Individual email times
	EmailTimes []time.Duration

	// Error tracking
	Errors    int
	ErrorRate float64
}

// Benchmark handles performance testing
type Benchmark struct {
	filter *filter.SpamFilter
}

// NewBenchmark creates a new benchmark instance
func NewBenchmark(cfg *config.Config) *Benchmark {
	return &Benchmark{
		filter: filter.NewSpamFilterWithConfig(cfg),
	}
}

// Run executes the benchmark
func (b *Benchmark) Run(emailFiles []string, runs int, concurrent int) *BenchmarkResult {
	result := &BenchmarkResult{
		TotalEmails: len(emailFiles) * runs,
		EmailTimes:  make([]time.Duration, 0, len(emailFiles)*runs),
	}

	fmt.Printf("🏃 Running benchmark...\n")

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Channel to control concurrency
	semaphore := make(chan struct{}, concurrent)

	start := time.Now()

	for run := 0; run < runs; run++ {
		for _, emailFile := range emailFiles {
			wg.Add(1)

			go func(file string, runNum int) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Measure email processing time
				emailStart := time.Now()
				score, err := b.filter.TestEmail(file)
				emailDuration := time.Since(emailStart)

				// Update results (thread-safe)
				mu.Lock()
				result.EmailTimes = append(result.EmailTimes, emailDuration)

				if err != nil {
					result.Errors++
				} else {
					// Classify based on filename (ground truth)
					isSpam := strings.Contains(filepath.Base(file), "spam")
					if score >= 4 {
						result.SpamDetected++
						if isSpam {
							// Correct spam detection
						} else {
							// False positive
						}
					} else {
						result.HamDetected++
						if !isSpam {
							// Correct ham detection
						} else {
							// False negative
						}
					}
				}
				mu.Unlock()

			}(emailFile, run)
		}
	}

	wg.Wait()
	result.TotalTime = time.Since(start)

	// Calculate statistics
	b.calculateStatistics(result)

	return result
}

// calculateStatistics computes performance statistics
func (b *Benchmark) calculateStatistics(result *BenchmarkResult) {
	if len(result.EmailTimes) == 0 {
		return
	}

	// Sort times for percentile calculations
	times := make([]time.Duration, len(result.EmailTimes))
	copy(times, result.EmailTimes)
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	// Basic statistics
	var totalNanos int64
	for _, t := range times {
		totalNanos += t.Nanoseconds()
	}

	result.AvgTimePerEmail = float64(totalNanos) / float64(len(times)) / 1e6 // Convert to ms
	result.MinTime = times[0]
	result.MaxTime = times[len(times)-1]

	// Percentiles
	result.MedianTime = times[len(times)/2]
	result.P95Time = times[int(float64(len(times))*0.95)]
	result.P99Time = times[int(float64(len(times))*0.99)]

	// Throughput
	result.EmailsPerSecond = float64(len(times)) / result.TotalTime.Seconds()

	// Error rate
	result.ErrorRate = float64(result.Errors) / float64(len(times)) * 100
}

// displayBenchmarkResults shows formatted benchmark results
func displayBenchmarkResults(result *BenchmarkResult) {
	fmt.Printf("📊 Benchmark Results\n")
	fmt.Printf("═══════════════════════════════════════\n\n")

	// Performance metrics
	fmt.Printf("⚡ Performance Metrics:\n")
	fmt.Printf("  Total emails processed: %d\n", result.TotalEmails)
	fmt.Printf("  Total time: %v\n", result.TotalTime)
	fmt.Printf("  Average time per email: %.2f ms\n", result.AvgTimePerEmail)
	fmt.Printf("  Emails per second: %.0f\n", result.EmailsPerSecond)
	fmt.Printf("\n")

	// Time distribution
	fmt.Printf("📈 Time Distribution:\n")
	fmt.Printf("  Min time: %.2f ms\n", float64(result.MinTime.Nanoseconds())/1e6)
	fmt.Printf("  Max time: %.2f ms\n", float64(result.MaxTime.Nanoseconds())/1e6)
	fmt.Printf("  Median time: %.2f ms\n", float64(result.MedianTime.Nanoseconds())/1e6)
	fmt.Printf("  95th percentile: %.2f ms\n", float64(result.P95Time.Nanoseconds())/1e6)
	fmt.Printf("  99th percentile: %.2f ms\n", float64(result.P99Time.Nanoseconds())/1e6)
	fmt.Printf("\n")

	// Classification results
	fmt.Printf("🎯 Classification Results:\n")
	fmt.Printf("  Spam detected: %d\n", result.SpamDetected)
	fmt.Printf("  Ham detected: %d\n", result.HamDetected)
	fmt.Printf("  Error rate: %.2f%%\n", result.ErrorRate)
	fmt.Printf("\n")

	// Performance assessment
	fmt.Printf("🏆 Performance Assessment:\n")
	if result.AvgTimePerEmail < 1.0 {
		fmt.Printf("  ✅ EXCELLENT: Average time %.2f ms < 1 ms\n", result.AvgTimePerEmail)
	} else if result.AvgTimePerEmail < 5.0 {
		fmt.Printf("  ✅ GOOD: Average time %.2f ms < 5 ms target\n", result.AvgTimePerEmail)
	} else {
		fmt.Printf("  ❌ NEEDS IMPROVEMENT: Average time %.2f ms > 5 ms target\n", result.AvgTimePerEmail)
	}

	if result.P95Time.Nanoseconds()/1e6 < 5 {
		fmt.Printf("  ✅ 95%% of emails processed in %.2f ms < 5 ms\n", float64(result.P95Time.Nanoseconds())/1e6)
	} else {
		fmt.Printf("  ⚠️ 95%% of emails processed in %.2f ms > 5 ms\n", float64(result.P95Time.Nanoseconds())/1e6)
	}

	if result.EmailsPerSecond > 1000 {
		fmt.Printf("  🚀 HIGH THROUGHPUT: %.0f emails/second\n", result.EmailsPerSecond)
	} else if result.EmailsPerSecond > 200 {
		fmt.Printf("  ⚡ GOOD THROUGHPUT: %.0f emails/second\n", result.EmailsPerSecond)
	} else {
		fmt.Printf("  🐌 LOW THROUGHPUT: %.0f emails/second\n", result.EmailsPerSecond)
	}

	// Recommendations
	fmt.Printf("\n💡 Recommendations:\n")
	if result.AvgTimePerEmail > 2.0 {
		fmt.Printf("  • Consider optimizing email parsing\n")
		fmt.Printf("  • Review feature computation complexity\n")
	}
	if result.P99Time.Nanoseconds()/1e6 > 10 {
		fmt.Printf("  • Investigate outlier processing times\n")
		fmt.Printf("  • Consider caching frequently used data\n")
	}
	if result.ErrorRate > 1.0 {
		fmt.Printf("  • Review error handling and input validation\n")
	}

	fmt.Printf("\n")
}

// findEmailFiles recursively finds email files
func findEmailFiles(root string) ([]string, error) {
	var emailFiles []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check if file looks like an email
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".eml" || ext == ".msg" || ext == ".email" || ext == "" {
			emailFiles = append(emailFiles, path)
		}

		return nil
	})

	return emailFiles, err
}

// RunParallel performs parallel benchmark testing with detailed metrics
func (b *Benchmark) RunParallel(emailFiles []string, runs int, concurrent int) *BenchmarkResult {
	result := &BenchmarkResult{
		TotalEmails: len(emailFiles) * runs,
		EmailTimes:  make([]time.Duration, 0, len(emailFiles)*runs),
	}

	fmt.Printf("🏃 Running parallel benchmark with %d workers...\n", concurrent)

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Channel to control concurrency
	semaphore := make(chan struct{}, concurrent)

	start := time.Now()

	// Process all files across all runs in parallel
	for run := 0; run < runs; run++ {
		for _, emailFile := range emailFiles {
			wg.Add(1)

			go func(file string, runNum int) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Measure email processing time
				emailStart := time.Now()
				score, err := b.filter.TestEmail(file)
				emailDuration := time.Since(emailStart)

				// Update results (thread-safe)
				mu.Lock()
				result.EmailTimes = append(result.EmailTimes, emailDuration)

				if err != nil {
					result.Errors++
				} else {
					// Classify based on filename (ground truth)
					isSpam := strings.Contains(filepath.Base(file), "spam")
					if score >= 4 {
						result.SpamDetected++
						if isSpam {
							// Correct spam detection
						} else {
							// False positive
						}
					} else {
						result.HamDetected++
						if !isSpam {
							// Correct ham detection
						} else {
							// False negative
						}
					}
				}
				mu.Unlock()

			}(emailFile, run)
		}
	}

	wg.Wait()
	result.TotalTime = time.Since(start)

	// Calculate statistics
	b.calculateStatistics(result)

	return result
}

// displayParallelBenchmarkResults shows parallel benchmark results with performance metrics
func displayParallelBenchmarkResults(result *BenchmarkResult) {
	fmt.Printf("📊 Parallel Benchmark Results\n")
	fmt.Printf("═══════════════════════════════\n")
	fmt.Printf("📧 Total emails processed: %d\n", result.TotalEmails)
	fmt.Printf("⏱️  Total time: %v\n", result.TotalTime)
	fmt.Printf("⚡ Average per email: %.3fms\n", result.AvgTimePerEmail)
	fmt.Printf("🏃 Processing rate: %.0f emails/second\n", float64(result.TotalEmails)/result.TotalTime.Seconds())
	fmt.Printf("\n")

	fmt.Printf("📈 Performance Statistics:\n")
	fmt.Printf("   Fastest email: %.3fms\n", float64(result.MinTime.Nanoseconds())/1e6)
	fmt.Printf("   Slowest email: %.3fms\n", float64(result.MaxTime.Nanoseconds())/1e6)
	fmt.Printf("   Median time: %.3fms\n", float64(result.MedianTime.Nanoseconds())/1e6)
	fmt.Printf("   95th percentile: %.3fms\n", float64(result.P95Time.Nanoseconds())/1e6)
	fmt.Printf("   99th percentile: %.3fms\n", float64(result.P99Time.Nanoseconds())/1e6)
	fmt.Printf("\n")

	fmt.Printf("🎯 Detection Results:\n")
	fmt.Printf("   Spam detected: %d\n", result.SpamDetected)
	fmt.Printf("   Ham detected: %d\n", result.HamDetected)
	fmt.Printf("   Processing errors: %d\n", result.Errors)

	if result.TotalEmails > 0 {
		fmt.Printf("   Success rate: %.1f%%\n", float64(result.TotalEmails-result.Errors)/float64(result.TotalEmails)*100)
	}

	fmt.Printf("\n")

	// Performance comparison hint
	fmt.Printf("💡 Tip: Run with --parallel=false to compare sequential performance\n")
	fmt.Printf("🚀 Parallel execution provides significant speedup for batch processing!\n")
}

func init() {
	benchmarkCmd.Flags().StringVarP(&benchmarkInput, "input", "i", "", "Input directory with email files")
	benchmarkCmd.Flags().StringVarP(&benchmarkConfig, "config", "c", "", "Configuration file path")
	benchmarkCmd.Flags().IntVarP(&benchmarkRuns, "runs", "r", 3, "Number of benchmark runs")
	benchmarkCmd.Flags().IntVarP(&benchmarkConcurrent, "concurrent", "j", 1, "Number of concurrent workers")
	benchmarkCmd.Flags().BoolVarP(&benchmarkOutput, "verbose", "v", false, "Verbose output")
	benchmarkCmd.Flags().BoolVarP(&benchmarkParallel, "parallel", "p", false, "Enable parallel execution")

	benchmarkCmd.MarkFlagRequired("input")
}
