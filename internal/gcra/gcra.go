// Package gcra implements the Generic Cell Rate Algorithm.
//
// GCRA is a rate limiting algorithm that provides smooth rate limiting
// without burst behavior. It's commonly used in networking for traffic
// shaping and is equivalent to a virtual scheduling algorithm.
package gcra

import (
	"sync"
	"time"
)

// GCRA implements the Generic Cell Rate Algorithm.
// It maintains a Theoretical Arrival Time (TAT) to determine
// whether requests conform to the desired rate.
type GCRA struct {
	mu       sync.Mutex
	rate     float64       // 每秒允许的请求数
	burst    int           // 最大突发大小
	emission time.Duration // 每个请求的发射间隔
	delay    time.Duration // 最大延迟容忍（burst * emission）
	tat      time.Time     // 理论到达时间
}

// NewGCRA creates a new GCRA rate limiter.
// rate specifies requests per second, burst specifies the maximum burst size.
func NewGCRA(rate float64, burst int) *GCRA {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}

	emission := time.Duration(float64(time.Second) / rate)
	delay := emission * time.Duration(burst)

	return &GCRA{
		rate:     rate,
		burst:    burst,
		emission: emission,
		delay:    delay,
	}
}

// Allow checks if a single request at the given time is allowed.
// Returns true if the request conforms to the rate limit.
func (g *GCRA) Allow(now time.Time) bool {
	return g.AllowN(now, 1)
}

// AllowN checks if n requests at the given time are allowed.
// Returns true if the requests conform to the rate limit.
func (g *GCRA) AllowN(now time.Time, n int) bool {
	if n <= 0 {
		return true
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// 计算本次请求需要的发射时间
	increment := g.emission * time.Duration(n)

	// 如果 TAT 为零，初始化为当前时间
	if g.tat.IsZero() {
		newTAT := now.Add(increment)
		// 即使第一次请求也要检查是否超过延迟容忍
		limit := now.Add(g.delay)
		if newTAT.After(limit) {
			return false
		}
		g.tat = newTAT
		return true
	}

	// 新的 TAT
	newTAT := g.tat.Add(increment)

	// 如果当前时间已经超过了 TAT，重新计算
	if now.After(g.tat) {
		newTAT = now.Add(increment)
	}

	// 检查是否超过了最大延迟容忍
	limit := now.Add(g.delay)
	if newTAT.After(limit) {
		return false
	}

	g.tat = newTAT
	return true
}

// Rate returns the configured rate (requests per second).
func (g *GCRA) Rate() float64 {
	return g.rate
}

// Burst returns the configured burst size.
func (g *GCRA) Burst() int {
	return g.burst
}

// Emission returns the emission interval between requests.
func (g *GCRA) Emission() time.Duration {
	return g.emission
}

// Reset resets the GCRA state, clearing the TAT.
func (g *GCRA) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tat = time.Time{}
}

// TAT returns the current Theoretical Arrival Time.
func (g *GCRA) TAT() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.tat
}
