package dns

import (
	"context"
	"fmt"
	"net"
	"sync"
)

// AsyncResult represents a DNS lookup result that may complete in the future
type AsyncResult struct {
	done   chan struct{}
	result interface{}
	err    error
	cached bool
}

// Wait blocks until the DNS lookup is complete and returns the result
func (ar *AsyncResult) Wait() (interface{}, error) {
	<-ar.done
	return ar.result, ar.err
}

// IsDone returns true if the DNS lookup has completed
func (ar *AsyncResult) IsDone() bool {
	select {
	case <-ar.done:
		return true
	default:
		return false
	}
}

// IsFromCache returns true if the result came from cache
func (ar *AsyncResult) IsFromCache() bool {
	return ar.cached
}

// AsyncClient provides non-blocking DNS operations with caching
type AsyncClient struct {
	*Client // Embed the synchronous client
	
	// Worker pool for async operations
	workers    int
	workQueue  chan func()
	stopChan   chan struct{}
	workerWG   sync.WaitGroup
	
	// Pending operations tracking
	pendingMu sync.RWMutex
	pending   map[string][]*AsyncResult
}

// NewAsyncClient creates a new async DNS client
func NewAsyncClient(config Config, workers int) *AsyncClient {
	if workers <= 0 {
		workers = 10 // Default worker count
	}
	
	client := &AsyncClient{
		Client:    NewClient(config),
		workers:   workers,
		workQueue: make(chan func(), workers*2), // Buffered queue
		stopChan:  make(chan struct{}),
		pending:   make(map[string][]*AsyncResult),
	}
	
	// Start worker goroutines
	client.startWorkers()
	
	return client
}

// startWorkers launches the worker goroutines
func (ac *AsyncClient) startWorkers() {
	for i := 0; i < ac.workers; i++ {
		ac.workerWG.Add(1)
		go ac.worker()
	}
}

// worker processes DNS lookup jobs
func (ac *AsyncClient) worker() {
	defer ac.workerWG.Done()
	
	for {
		select {
		case job := <-ac.workQueue:
			job()
		case <-ac.stopChan:
			return
		}
	}
}

// Stop gracefully shuts down the async client
func (ac *AsyncClient) Stop() {
	close(ac.stopChan)
	ac.workerWG.Wait()
	close(ac.workQueue)
}

// LookupTXTAsync performs asynchronous TXT record lookup
func (ac *AsyncClient) LookupTXTAsync(domain string) *AsyncResult {
	cacheKey := fmt.Sprintf("TXT:%s", domain)
	
	// Check cache first
	if ac.config.EnableCaching {
		if cached := ac.getFromCache(cacheKey); cached != nil {
			// Return immediately with cached result
			result := &AsyncResult{
				done:   make(chan struct{}),
				result: cached,
				err:    nil,
				cached: true,
			}
			close(result.done)
			
			ac.mu.Lock()
			ac.stats.Hits++
			ac.mu.Unlock()
			
			return result
		}
	}
	
	// Check if there's already a pending request for this domain
	ac.pendingMu.Lock()
	if existingResults, exists := ac.pending[cacheKey]; exists {
		// Create a new result that will be notified when the pending request completes
		result := &AsyncResult{
			done:   make(chan struct{}),
			cached: false,
		}
		ac.pending[cacheKey] = append(existingResults, result)
		ac.pendingMu.Unlock()
		return result
	}
	
	// Create new pending request
	result := &AsyncResult{
		done:   make(chan struct{}),
		cached: false,
	}
	ac.pending[cacheKey] = []*AsyncResult{result}
	ac.pendingMu.Unlock()
	
	// Submit job to worker pool
	select {
	case ac.workQueue <- func() {
		ac.performTXTLookup(cacheKey, domain)
	}:
		// Job queued successfully
	default:
		// Queue is full, perform lookup synchronously to avoid blocking
		go ac.performTXTLookup(cacheKey, domain)
	}
	
	return result
}

// performTXTLookup executes the actual TXT lookup and notifies all waiting results
func (ac *AsyncClient) performTXTLookup(cacheKey, domain string) {
	// Perform the actual DNS lookup
	ctx := context.Background()
	records, err := ac.resolver.LookupTXT(ctx, domain)
	
	// Update stats
	ac.mu.Lock()
	if err != nil {
		ac.stats.Errors++
	} else {
		ac.stats.Misses++
	}
	ac.mu.Unlock()
	
	// Cache the result if successful
	if err == nil && ac.config.EnableCaching {
		ac.setInCache(cacheKey, records)
	}
	
	// Notify all waiting results
	ac.pendingMu.Lock()
	results := ac.pending[cacheKey]
	delete(ac.pending, cacheKey)
	ac.pendingMu.Unlock()
	
	for _, result := range results {
		result.result = records
		result.err = err
		close(result.done)
	}
}

// LookupAAsync performs asynchronous A record lookup
func (ac *AsyncClient) LookupAAsync(domain string) *AsyncResult {
	cacheKey := fmt.Sprintf("A:%s", domain)
	
	// Check cache first
	if ac.config.EnableCaching {
		if cached := ac.getFromCache(cacheKey); cached != nil {
			result := &AsyncResult{
				done:   make(chan struct{}),
				result: cached,
				err:    nil,
				cached: true,
			}
			close(result.done)
			
			ac.mu.Lock()
			ac.stats.Hits++
			ac.mu.Unlock()
			
			return result
		}
	}
	
	// Handle pending requests
	ac.pendingMu.Lock()
	if existingResults, exists := ac.pending[cacheKey]; exists {
		result := &AsyncResult{
			done:   make(chan struct{}),
			cached: false,
		}
		ac.pending[cacheKey] = append(existingResults, result)
		ac.pendingMu.Unlock()
		return result
	}
	
	result := &AsyncResult{
		done:   make(chan struct{}),
		cached: false,
	}
	ac.pending[cacheKey] = []*AsyncResult{result}
	ac.pendingMu.Unlock()
	
	// Submit job
	select {
	case ac.workQueue <- func() {
		ac.performALookup(cacheKey, domain)
	}:
	default:
		go ac.performALookup(cacheKey, domain)
	}
	
	return result
}

// performALookup executes the actual A lookup
func (ac *AsyncClient) performALookup(cacheKey, domain string) {
	ctx := context.Background()
	ipAddrs, err := ac.resolver.LookupIPAddr(ctx, domain)
	
	ac.mu.Lock()
	if err != nil {
		ac.stats.Errors++
	} else {
		ac.stats.Misses++
	}
	ac.mu.Unlock()
	
	// Extract IPv4 addresses
	var ips []net.IP
	if err == nil {
		for _, addr := range ipAddrs {
			if ipv4 := addr.IP.To4(); ipv4 != nil {
				ips = append(ips, ipv4)
			}
		}
		
		if ac.config.EnableCaching {
			ac.setInCache(cacheKey, ips)
		}
	}
	
	// Notify waiters
	ac.pendingMu.Lock()
	results := ac.pending[cacheKey]
	delete(ac.pending, cacheKey)
	ac.pendingMu.Unlock()
	
	for _, result := range results {
		result.result = ips
		result.err = err
		close(result.done)
	}
}

// LookupMXAsync performs asynchronous MX record lookup
func (ac *AsyncClient) LookupMXAsync(domain string) *AsyncResult {
	cacheKey := fmt.Sprintf("MX:%s", domain)
	
	// Check cache first
	if ac.config.EnableCaching {
		if cached := ac.getFromCache(cacheKey); cached != nil {
			result := &AsyncResult{
				done:   make(chan struct{}),
				result: cached,
				err:    nil,
				cached: true,
			}
			close(result.done)
			
			ac.mu.Lock()
			ac.stats.Hits++
			ac.mu.Unlock()
			
			return result
		}
	}
	
	// Handle pending requests
	ac.pendingMu.Lock()
	if existingResults, exists := ac.pending[cacheKey]; exists {
		result := &AsyncResult{
			done:   make(chan struct{}),
			cached: false,
		}
		ac.pending[cacheKey] = append(existingResults, result)
		ac.pendingMu.Unlock()
		return result
	}
	
	result := &AsyncResult{
		done:   make(chan struct{}),
		cached: false,
	}
	ac.pending[cacheKey] = []*AsyncResult{result}
	ac.pendingMu.Unlock()
	
	// Submit job
	select {
	case ac.workQueue <- func() {
		ac.performMXLookup(cacheKey, domain)
	}:
	default:
		go ac.performMXLookup(cacheKey, domain)
	}
	
	return result
}

// performMXLookup executes the actual MX lookup
func (ac *AsyncClient) performMXLookup(cacheKey, domain string) {
	ctx := context.Background()
	mxRecords, err := ac.resolver.LookupMX(ctx, domain)
	
	ac.mu.Lock()
	if err != nil {
		ac.stats.Errors++
	} else {
		ac.stats.Misses++
	}
	ac.mu.Unlock()
	
	if err == nil && ac.config.EnableCaching {
		ac.setInCache(cacheKey, mxRecords)
	}
	
	// Notify waiters
	ac.pendingMu.Lock()
	results := ac.pending[cacheKey]
	delete(ac.pending, cacheKey)
	ac.pendingMu.Unlock()
	
	for _, result := range results {
		result.result = mxRecords
		result.err = err
		close(result.done)
	}
}

// GetSPFRecordAsync retrieves SPF record asynchronously
func (ac *AsyncClient) GetSPFRecordAsync(domain string) *AsyncResult {
	txtResult := ac.LookupTXTAsync(domain)
	
	// If from cache, process immediately
	if txtResult.IsFromCache() {
		records, err := txtResult.Wait()
		if err != nil {
			result := &AsyncResult{
				done:   make(chan struct{}),
				result: "",
				err:    err,
				cached: true,
			}
			close(result.done)
			return result
		}
		
		txtRecords := records.([]string)
		for _, record := range txtRecords {
			if len(record) > 7 && record[:7] == "v=spf1 " {
				result := &AsyncResult{
					done:   make(chan struct{}),
					result: record,
					err:    nil,
					cached: true,
				}
				close(result.done)
				return result
			}
		}
		
		result := &AsyncResult{
			done:   make(chan struct{}),
			result: "",
			err:    fmt.Errorf("no SPF record found for domain %s", domain),
			cached: true,
		}
		close(result.done)
		return result
	}
	
	// Create async result that will process TXT records when ready
	result := &AsyncResult{
		done:   make(chan struct{}),
		cached: false,
	}
	
	go func() {
		records, err := txtResult.Wait()
		if err != nil {
			result.result = ""
			result.err = err
			close(result.done)
			return
		}
		
		txtRecords := records.([]string)
		for _, record := range txtRecords {
			if len(record) > 7 && record[:7] == "v=spf1 " {
				result.result = record
				result.err = nil
				close(result.done)
				return
			}
		}
		
		result.result = ""
		result.err = fmt.Errorf("no SPF record found for domain %s", domain)
		close(result.done)
	}()
	
	return result
}

// GetDMARCRecordAsync retrieves DMARC record asynchronously
func (ac *AsyncClient) GetDMARCRecordAsync(domain string) *AsyncResult {
	dmarcDomain := "_dmarc." + domain
	txtResult := ac.LookupTXTAsync(dmarcDomain)
	
	// If from cache, process immediately
	if txtResult.IsFromCache() {
		records, err := txtResult.Wait()
		if err != nil {
			result := &AsyncResult{
				done:   make(chan struct{}),
				result: "",
				err:    err,
				cached: true,
			}
			close(result.done)
			return result
		}
		
		txtRecords := records.([]string)
		for _, record := range txtRecords {
			if len(record) > 8 && record[:8] == "v=DMARC1" {
				result := &AsyncResult{
					done:   make(chan struct{}),
					result: record,
					err:    nil,
					cached: true,
				}
				close(result.done)
				return result
			}
		}
		
		result := &AsyncResult{
			done:   make(chan struct{}),
			result: "",
			err:    fmt.Errorf("no DMARC record found for domain %s", domain),
			cached: true,
		}
		close(result.done)
		return result
	}
	
	// Create async result that will process TXT records when ready
	result := &AsyncResult{
		done:   make(chan struct{}),
		cached: false,
	}
	
	go func() {
		records, err := txtResult.Wait()
		if err != nil {
			result.result = ""
			result.err = err
			close(result.done)
			return
		}
		
		txtRecords := records.([]string)
		for _, record := range txtRecords {
			if len(record) > 8 && record[:8] == "v=DMARC1" {
				result.result = record
				result.err = nil
				close(result.done)
				return
			}
		}
		
		result.result = ""
		result.err = fmt.Errorf("no DMARC record found for domain %s", domain)
		close(result.done)
	}()
	
	return result
}

// WaitForMultiple waits for multiple async results to complete
func WaitForMultiple(results ...*AsyncResult) []interface{} {
	responses := make([]interface{}, len(results))
	var wg sync.WaitGroup
	
	for i, result := range results {
		wg.Add(1)
		go func(idx int, res *AsyncResult) {
			defer wg.Done()
			value, err := res.Wait()
			if err != nil {
				responses[idx] = err
			} else {
				responses[idx] = value
			}
		}(i, result)
	}
	
	wg.Wait()
	return responses
} 