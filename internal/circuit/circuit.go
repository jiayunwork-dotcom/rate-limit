// Package circuit implements the circuit breaker pattern.
//
// A circuit breaker monitors for failures and opens the circuit
// (blocking requests) when the failure threshold is reached.
// After a timeout period, it enters a half-open state to test
// if the underlying issue has been resolved.
package circuit

import (
	"sync"
	"time"
)

// 断路器状态常量
const (
	StateClosed   = "closed"
	StateOpen     = "open"
	StateHalfOpen = "half-open"
)

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	mu             sync.Mutex
	state          string
	failures       int
	successes      int
	threshold      int
	timeout        time.Duration
	lastFailure    time.Time
	halfOpenMax    int
	clock          func() time.Time
}

// NewBreaker creates a new circuit breaker.
// threshold is the number of consecutive failures before opening the circuit.
// timeout is the duration to wait before transitioning from open to half-open.
func NewBreaker(threshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		state:       StateClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 1,
		clock:       time.Now,
	}
}

// NewBreakerWithClock creates a circuit breaker with a custom clock for testing.
func NewBreakerWithClock(threshold int, timeout time.Duration, clock func() time.Time) *Breaker {
	return &Breaker{
		state:       StateClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 1,
		clock:       clock,
	}
}

// Allow checks if a request is allowed through the circuit breaker.
// Returns true if the circuit is closed or half-open (for testing).
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否超时，应该转为半开状态
		now := b.clock()
		if now.Sub(b.lastFailure) >= b.timeout {
			b.state = StateHalfOpen
			b.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		// 半开状态允许有限的请求通过
		return b.successes < b.halfOpenMax
	default:
		return false
	}
}

// RecordSuccess records a successful request.
// In half-open state, a success closes the circuit.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.halfOpenMax {
			b.state = StateClosed
			b.failures = 0
			b.successes = 0
		}
	case StateClosed:
		b.failures = 0
	}
}

// RecordFailure records a failed request.
// If failures reach the threshold, the circuit opens.
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	applyFail(b)
}

// State returns the current state of the circuit breaker.
func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Failures returns the current failure count.
func (b *Breaker) Failures() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failures
}

// Reset resets the circuit breaker to closed state.
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.lastFailure = time.Time{}
}
