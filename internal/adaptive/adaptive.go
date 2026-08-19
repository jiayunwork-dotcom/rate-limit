// Package adaptive implements an adaptive rate limiter that adjusts
// its rate based on the observed error rate.
//
// When errors increase, the rate is reduced to give the system time
// to recover. When errors decrease, the rate gradually increases
// back toward the maximum.
package adaptive

import (
	"sync"
	"time"
)

// Limiter implements adaptive rate limiting that adjusts based on error rates.
type Limiter struct {
	mu          sync.Mutex
	baseRate    float64 // 基础速率
	minRate     float64 // 最小速率
	maxRate     float64 // 最大速率
	currentRate float64 // 当前速率
	tokens      float64 // 可用令牌数
	lastRefill  time.Time
	successes   int64
	errors      int64
	adjustEvery int64 // 每多少次请求调整一次
	clock       func() time.Time
}

// NewAdaptive creates a new adaptive rate limiter.
// baseRate is the initial rate (requests per second).
// minRate and maxRate define the bounds for rate adjustment.
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

// NewAdaptiveWithClock creates an adaptive limiter with a custom clock for testing.
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

// refill adds tokens based on elapsed time.
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

// adjust recalculates the current rate based on error rates.
func (l *Limiter) adjust() {
	total := l.successes + l.errors
	if total < l.adjustEvery {
		return
	}

	errorRate := float64(l.errors) / float64(total)

	// 根据错误率调整速率
	if errorRate > 0.5 {
		// 高错误率，大幅降低速率
		l.currentRate *= 0.5
	} else if errorRate > 0.2 {
		// 中等错误率，适度降低
		l.currentRate *= 0.8
	} else if errorRate < 0.05 {
		// 低错误率，逐步提高
		l.currentRate *= 1.1
	}

	// 确保在范围内
	if l.currentRate < l.minRate {
		l.currentRate = l.minRate
	}
	if l.currentRate > l.maxRate {
		l.currentRate = l.maxRate
	}

	// 重置计数器
	l.successes = 0
	l.errors = 0
}

// Allow checks if a request is allowed at the current adaptive rate.
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

// RecordSuccess records a successful request, potentially increasing the rate.
func (l *Limiter) RecordSuccess() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.successes++
	l.adjust()
}

// RecordError records a failed request, potentially decreasing the rate.
func (l *Limiter) RecordError() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors++
	l.adjust()
}

// CurrentRate returns the current adaptive rate.
func (l *Limiter) CurrentRate() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentRate
}

// SetAdjustInterval sets how many requests trigger a rate adjustment.
func (l *Limiter) SetAdjustInterval(n int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.adjustEvery = n
}

// Reset resets the limiter to its base rate.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentRate = l.baseRate
	l.tokens = l.baseRate
	l.successes = 0
	l.errors = 0
}
