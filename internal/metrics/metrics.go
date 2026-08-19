// Package metrics provides rate limiter metrics collection.
//
// It tracks allow/deny counts per key and provides aggregated
// statistics for monitoring rate limiter effectiveness.
package metrics

import (
	"sync"
)

// Stats holds the statistics for a single rate limiting key.
type Stats struct {
	Allowed  int64   // 允许的请求数
	Denied   int64   // 拒绝的请求数
	Total    int64   // 总请求数
	DenyRate float64 // 拒绝率
}

// Collector collects rate limiter metrics.
type Collector struct {
	mu      sync.Mutex
	allowed map[string]int64
	denied  map[string]int64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		allowed: make(map[string]int64),
		denied:  make(map[string]int64),
	}
}

// RecordAllow records that a request for the given key was allowed.
func (c *Collector) RecordAllow(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowed[key]++
}

// RecordDeny records that a request for the given key was denied.
func (c *Collector) RecordDeny(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.denied[key]++
}

// Stats returns the statistics for all tracked keys.
func (c *Collector) Stats() map[string]Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]Stats)

	// 收集所有key
	keys := make(map[string]struct{})
	for k := range c.allowed {
		keys[k] = struct{}{}
	}
	for k := range c.denied {
		keys[k] = struct{}{}
	}

	for key := range keys {
		allowed := c.allowed[key]
		denied := c.denied[key]
		total := allowed + denied
		var denyRate float64
		if total > 0 {
			denyRate = float64(denied) / float64(total)
		}
		result[key] = Stats{
			Allowed:  allowed,
			Denied:   denied,
			Total:    total,
			DenyRate: denyRate,
		}
	}

	return result
}

// StatsFor returns the statistics for a specific key.
func (c *Collector) StatsFor(key string) Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	allowed := c.allowed[key]
	denied := c.denied[key]
	total := allowed + denied
	var denyRate float64
	if total > 0 {
		denyRate = float64(denied) / float64(total)
	}
	return Stats{
		Allowed:  allowed,
		Denied:   denied,
		Total:    total,
		DenyRate: denyRate,
	}
}

// Reset clears all collected metrics.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowed = make(map[string]int64)
	c.denied = make(map[string]int64)
}

// Keys returns all tracked keys.
func (c *Collector) Keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make(map[string]struct{})
	for k := range c.allowed {
		keys[k] = struct{}{}
	}
	for k := range c.denied {
		keys[k] = struct{}{}
	}
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}
