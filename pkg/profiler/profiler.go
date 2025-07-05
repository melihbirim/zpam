package profiler

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Profiler tracks execution times for different operations
type Profiler struct {
	mu    sync.RWMutex
	times map[string][]time.Duration
}

// NewProfiler creates a new profiler
func NewProfiler() *Profiler {
	return &Profiler{
		times: make(map[string][]time.Duration),
	}
}

// Timer represents a timing operation
type Timer struct {
	profiler *Profiler
	name     string
	start    time.Time
}

// Start begins timing an operation
func (p *Profiler) Start(name string) *Timer {
	return &Timer{
		profiler: p,
		name:     name,
		start:    time.Now(),
	}
}

// Stop completes the timing and records the duration
func (t *Timer) Stop() time.Duration {
	duration := time.Since(t.start)
	
	t.profiler.mu.Lock()
	t.profiler.times[t.name] = append(t.profiler.times[t.name], duration)
	t.profiler.mu.Unlock()
	
	return duration
}

// Record manually records a timing
func (p *Profiler) Record(name string, duration time.Duration) {
	p.mu.Lock()
	p.times[name] = append(p.times[name], duration)
	p.mu.Unlock()
}

// Stats contains timing statistics
type Stats struct {
	Name     string
	Count    int
	Total    time.Duration
	Average  time.Duration
	Min      time.Duration
	Max      time.Duration
	Median   time.Duration
	P95      time.Duration
	P99      time.Duration
}

// GetStats returns timing statistics for an operation
func (p *Profiler) GetStats(name string) *Stats {
	p.mu.RLock()
	times, exists := p.times[name]
	p.mu.RUnlock()
	
	if !exists || len(times) == 0 {
		return &Stats{Name: name, Count: 0}
	}
	
	// Create a copy and sort
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	// Calculate statistics
	var total time.Duration
	for _, t := range sorted {
		total += t
	}
	
	stats := &Stats{
		Name:    name,
		Count:   len(sorted),
		Total:   total,
		Average: total / time.Duration(len(sorted)),
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		Median:  sorted[len(sorted)/2],
	}
	
	if len(sorted) > 1 {
		stats.P95 = sorted[int(float64(len(sorted))*0.95)]
		stats.P99 = sorted[int(float64(len(sorted))*0.99)]
	}
	
	return stats
}

// GetAllStats returns statistics for all tracked operations
func (p *Profiler) GetAllStats() []*Stats {
	p.mu.RLock()
	names := make([]string, 0, len(p.times))
	for name := range p.times {
		names = append(names, name)
	}
	p.mu.RUnlock()
	
	sort.Strings(names)
	
	stats := make([]*Stats, 0, len(names))
	for _, name := range names {
		stats = append(stats, p.GetStats(name))
	}
	
	return stats
}

// Reset clears all timing data
func (p *Profiler) Reset() {
	p.mu.Lock()
	p.times = make(map[string][]time.Duration)
	p.mu.Unlock()
}

// PrintReport prints a formatted timing report
func (p *Profiler) PrintReport() {
	stats := p.GetAllStats()
	
	if len(stats) == 0 {
		fmt.Println("No timing data available")
		return
	}
	
	fmt.Printf("⏱️  Performance Profile Report\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("%-20s %8s %10s %8s %8s %8s %8s %8s\n", 
		"Operation", "Count", "Total", "Avg", "Min", "Max", "P95", "P99")
	fmt.Printf("─────────────────────────────────────────────────────────────────\n")
	
	for _, stat := range stats {
		if stat.Count == 0 {
			continue
		}
		
		fmt.Printf("%-20s %8d %10s %8s %8s %8s %8s %8s\n",
			truncate(stat.Name, 20),
			stat.Count,
			formatDuration(stat.Total),
			formatDuration(stat.Average),
			formatDuration(stat.Min),
			formatDuration(stat.Max),
			formatDuration(stat.P95),
			formatDuration(stat.P99),
		)
	}
	
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	} else {
		return fmt.Sprintf("%.3fs", d.Seconds())
	}
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ProfiledFunction is a helper to profile function execution
func (p *Profiler) ProfiledFunction(name string, fn func()) time.Duration {
	timer := p.Start(name)
	fn()
	return timer.Stop()
}

// Global profiler instance
var globalProfiler = NewProfiler()

// Start starts a global timer
func Start(name string) *Timer {
	return globalProfiler.Start(name)
}

// Record records a timing globally
func Record(name string, duration time.Duration) {
	globalProfiler.Record(name, duration)
}

// GetStats gets global stats
func GetStats(name string) *Stats {
	return globalProfiler.GetStats(name)
}

// PrintReport prints global timing report
func PrintReport() {
	globalProfiler.PrintReport()
}

// Reset resets global profiler
func Reset() {
	globalProfiler.Reset()
} 