package priority

import (
	"sync"
	"time"
)

type PriorityLimiter struct {
	mu       sync.Mutex
	levels   int
	rates    []float64
	tokens   []float64
	lastTime time.Time
	clock    func() time.Time
}

func NewPriority(levels int, baseRate float64) *PriorityLimiter {
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
		lastTime: time.Now(),
		clock:    time.Now,
	}
}

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

func (p *PriorityLimiter) SetRate(priority int, rate float64) {
	if priority < 0 || priority >= p.levels {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.rates[priority] = rate
}

func (p *PriorityLimiter) Rate(priority int) float64 {
	if priority < 0 || priority >= p.levels {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rates[priority]
}

func (p *PriorityLimiter) Levels() int {
	return p.levels
}
