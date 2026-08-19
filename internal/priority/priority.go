// Package priority implements priority-based rate limiting.
//
// Higher priority requests get higher rate limits, allowing
// important traffic to flow even when lower priority requests
// are being throttled.
package priority

import (
	"sync"
	"time"
)

// PriorityLimiter implements priority-based rate limiting.
// Each priority level has its own rate allocation.
type PriorityLimiter struct {
	mu       sync.Mutex
	levels   int
	rates    []float64
	tokens   []float64
	lastTime time.Time
	clock    func() time.Time
}

// NewPriority creates a new priority-based rate limiter.
// levels specifies the number of priority levels (0 = highest priority).
// baseRate is the rate for the lowest priority level.
// Higher priorities get proportionally higher rates.
func NewPriority(levels int, baseRate float64) *PriorityLimiter {
	if levels <= 0 {
		levels = 1
	}

	rates := make([]float64, levels)
	tokens := make([]float64, levels)
	for i := 0; i < levels; i++ {
		// 高优先级（低索引）获得更高的速率
		multiplier := float64(levels - i)
		rates[i] = baseRate * multiplier
		tokens[i] = rates[i]
	}

	return &PriorityLimiter{
		levels:   levels,
		rates:    rates,
		tokens:   tokens,
		lastTime: time.Now(),
		clock:    time.Now,
	}
}

// NewPriorityWithClock creates a priority limiter with a custom clock for testing.
func NewPriorityWithClock(levels int, baseRate float64, clock func() time.Time) *PriorityLimiter {
	if levels <= 0 {
		levels = 1
	}

	rates := make([]float64, levels)
	tokens := make([]float64, levels)
	for i := 0; i < levels; i++ {
		multiplier := float64(levels - i)
		rates[i] = baseRate * multiplier
		tokens[i] = rates[i]
	}

	return &PriorityLimiter{
		levels:   levels,
		rates:    rates,
		tokens:   tokens,
		lastTime: clock(),
		clock:    clock,
	}
}

// refill adds tokens to all priority levels based on elapsed time.
func (p *PriorityLimiter) refill() {
	now := p.clock()
	elapsed := now.Sub(p.lastTime).Seconds()
	if elapsed <= 0 {
		return
	}

	for i := range p.tokens {
		p.tokens[i] += elapsed * p.rates[i]
		if p.tokens[i] > p.rates[i] {
			p.tokens[i] = p.rates[i]
		}
	}
	p.lastTime = now
}

// Allow checks if a request with the given priority level is allowed.
// Priority 0 is the highest priority. Returns false if the priority
// is invalid or no tokens are available.
func (p *PriorityLimiter) Allow(priority int) bool {
	if priority < 0 || priority >= p.levels {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.refill()

	if p.tokens[priority] < 1 {
		return false
	}
	p.tokens[priority]--
	return true
}

// SetRate sets the rate for a specific priority level.
func (p *PriorityLimiter) SetRate(priority int, rate float64) {
	if priority < 0 || priority >= p.levels {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.rates[priority] = rate
}

// Rate returns the current rate for a priority level.
func (p *PriorityLimiter) Rate(priority int) float64 {
	if priority < 0 || priority >= p.levels {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rates[priority]
}

// Levels returns the number of priority levels.
func (p *PriorityLimiter) Levels() int {
	return p.levels
}
