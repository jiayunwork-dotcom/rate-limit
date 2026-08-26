package adaptive

import (
	"sync"
	"time"
)

type Limiter struct {
	mu          sync.Mutex
	baseRate    float64
	minRate     float64
	maxRate     float64
	currentRate float64
	tokens      float64
	lastRefill  time.Time
	successes   int64
	errors      int64
	adjustEvery int64
	clock       func() time.Time
}

func NewAdaptive(baseRate float64, minRate, maxRate float64) *Limiter {
	return &Limiter{
		baseRate:    baseRate,
		minRate:     minRate,
		maxRate:     maxRate,
		currentRate: baseRate,
		tokens:      baseRate,
		lastRefill:  time.Now(),
		adjustEvery: 10,
		clock:       time.Now,
	}
}

func NewAdaptiveWithClock(baseRate, minRate, maxRate float64, clock func() time.Time) *Limiter {
	return &Limiter{
		baseRate:    baseRate,
		minRate:     minRate,
		maxRate:     maxRate,
		currentRate: baseRate,
		tokens:      baseRate,
		lastRefill:  clock(),
		adjustEvery: 10,
		clock:       clock,
	}
}

func (l *Limiter) refill() {
	now := l.clock()
	elapsed := now.Sub(l.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	l.tokens += elapsed * l.currentRate
	if l.tokens > l.currentRate {
		l.tokens = l.currentRate
	}
	l.lastRefill = now
}

func (l *Limiter) adjust() {
	total := l.successes + l.errors
	if total < l.adjustEvery {
		return
	}

	errorRate := float64(l.errors) / float64(total)

	if errorRate > 0.5 {
		l.currentRate *= 0.5
	} else if errorRate > 0.2 {
		l.currentRate *= 0.8
	} else if errorRate < 0.05 {
		l.currentRate *= 1.1
	}

	if l.currentRate < l.minRate {
		l.currentRate = l.minRate
	}
	if l.currentRate > l.maxRate {
		l.currentRate = l.maxRate
	}

	l.successes = 0
	l.errors = 0
}

func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func (l *Limiter) RecordSuccess() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.successes++
	l.adjust()
}

func (l *Limiter) RecordError() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors++
	l.adjust()
}

func (l *Limiter) CurrentRate() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentRate
}

func (l *Limiter) SetAdjustInterval(n int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.adjustEvery = n
}

func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentRate = l.baseRate
	l.tokens = l.baseRate
	l.successes = 0
	l.errors = 0
}
