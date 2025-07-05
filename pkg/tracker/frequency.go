package tracker

import (
	"strings"
	"sync"
	"time"
)

// FrequencyTracker tracks sender frequency and patterns
type FrequencyTracker struct {
	mu      sync.RWMutex
	senders map[string]*SenderStats
	
	// Configuration
	windowMinutes int
	maxCacheSize  int
}

// SenderStats tracks statistics for a sender
type SenderStats struct {
	Email        string
	Domain       string
	TotalEmails  int
	RecentEmails []time.Time
	FirstSeen    time.Time
	LastSeen     time.Time
	
	// Pattern analysis
	SuspiciousPatterns int
	SpamScore         float64
}

// FrequencyResult contains frequency analysis results
type FrequencyResult struct {
	IsFrequentSender bool
	EmailsInWindow   int
	SuspiciousRatio  float64
	FrequencyScore   float64
}

// NewFrequencyTracker creates a new frequency tracker
func NewFrequencyTracker(windowMinutes, maxCacheSize int) *FrequencyTracker {
	return &FrequencyTracker{
		senders:       make(map[string]*SenderStats),
		windowMinutes: windowMinutes,
		maxCacheSize:  maxCacheSize,
	}
}

// TrackSender tracks an email sender
func (ft *FrequencyTracker) TrackSender(email, domain string, isSpam bool) *FrequencyResult {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	
	now := time.Now()
	email = strings.ToLower(email)
	domain = strings.ToLower(domain)
	
	// Get or create sender stats
	stats, exists := ft.senders[email]
	if !exists {
		stats = &SenderStats{
			Email:       email,
			Domain:      domain,
			FirstSeen:   now,
			RecentEmails: make([]time.Time, 0),
		}
		ft.senders[email] = stats
	}
	
	// Update stats
	stats.TotalEmails++
	stats.LastSeen = now
	stats.RecentEmails = append(stats.RecentEmails, now)
	
	if isSpam {
		stats.SuspiciousPatterns++
	}
	
	// Clean old entries from recent emails
	windowStart := now.Add(-time.Duration(ft.windowMinutes) * time.Minute)
	recentCount := 0
	for i := len(stats.RecentEmails) - 1; i >= 0; i-- {
		if stats.RecentEmails[i].After(windowStart) {
			recentCount++
		} else {
			stats.RecentEmails = stats.RecentEmails[i+1:]
			break
		}
	}
	
	// Calculate frequency metrics
	result := &FrequencyResult{
		EmailsInWindow: recentCount,
	}
	
	// Determine if frequent sender (more than 5 emails in window)
	result.IsFrequentSender = recentCount > 5
	
	// Calculate suspicious ratio
	if stats.TotalEmails > 0 {
		result.SuspiciousRatio = float64(stats.SuspiciousPatterns) / float64(stats.TotalEmails)
	}
	
	// Calculate frequency score based on patterns
	result.FrequencyScore = ft.calculateFrequencyScore(stats, recentCount)
	
	// Clean cache if too large
	if len(ft.senders) > ft.maxCacheSize {
		ft.cleanOldEntries()
	}
	
	return result
}

// calculateFrequencyScore calculates a spam score based on frequency patterns
func (ft *FrequencyTracker) calculateFrequencyScore(stats *SenderStats, recentCount int) float64 {
	var score float64
	
	// High frequency in short time (bulk sending)
	if recentCount > 20 {
		score += 4.0
	} else if recentCount > 10 {
		score += 2.0
	} else if recentCount > 5 {
		score += 1.0
	}
	
	// High suspicious ratio
	suspiciousRatio := float64(stats.SuspiciousPatterns) / float64(stats.TotalEmails)
	if suspiciousRatio > 0.8 {
		score += 3.0
	} else if suspiciousRatio > 0.5 {
		score += 2.0
	} else if suspiciousRatio > 0.3 {
		score += 1.0
	}
	
	// New sender with high volume (suspicious)
	timeSinceFirst := time.Since(stats.FirstSeen)
	if timeSinceFirst < time.Hour && recentCount > 5 {
		score += 2.0
	}
	
	// Very short intervals between emails
	if len(stats.RecentEmails) > 1 {
		avgInterval := timeSinceFirst / time.Duration(len(stats.RecentEmails)-1)
		if avgInterval < 30*time.Second {
			score += 2.0
		} else if avgInterval < 2*time.Minute {
			score += 1.0
		}
	}
	
	return score
}

// GetSenderStats returns statistics for a sender
func (ft *FrequencyTracker) GetSenderStats(email string) *SenderStats {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	
	email = strings.ToLower(email)
	if stats, exists := ft.senders[email]; exists {
		// Return a copy to avoid race conditions
		statsCopy := *stats
		return &statsCopy
	}
	
	return nil
}

// GetDomainStats returns aggregated statistics for a domain
func (ft *FrequencyTracker) GetDomainStats(domain string) map[string]*SenderStats {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	
	domain = strings.ToLower(domain)
	domainStats := make(map[string]*SenderStats)
	
	for email, stats := range ft.senders {
		if stats.Domain == domain {
			statsCopy := *stats
			domainStats[email] = &statsCopy
		}
	}
	
	return domainStats
}

// cleanOldEntries removes old entries to keep cache size manageable
func (ft *FrequencyTracker) cleanOldEntries() {
	now := time.Now()
	cutoff := now.Add(-24 * time.Hour) // Remove entries older than 24 hours
	
	for email, stats := range ft.senders {
		if stats.LastSeen.Before(cutoff) {
			delete(ft.senders, email)
		}
	}
}

// GetStats returns overall tracker statistics
func (ft *FrequencyTracker) GetStats() map[string]interface{} {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	
	totalSenders := len(ft.senders)
	activeSenders := 0
	suspiciousSenders := 0
	
	now := time.Now()
	windowStart := now.Add(-time.Duration(ft.windowMinutes) * time.Minute)
	
	for _, stats := range ft.senders {
		if stats.LastSeen.After(windowStart) {
			activeSenders++
		}
		
		suspiciousRatio := float64(stats.SuspiciousPatterns) / float64(stats.TotalEmails)
		if suspiciousRatio > 0.5 {
			suspiciousSenders++
		}
	}
	
	return map[string]interface{}{
		"total_senders":      totalSenders,
		"active_senders":     activeSenders,
		"suspicious_senders": suspiciousSenders,
		"window_minutes":     ft.windowMinutes,
	}
}

// Reset clears all tracking data
func (ft *FrequencyTracker) Reset() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	
	ft.senders = make(map[string]*SenderStats)
} 