package circuit

import (
	"sync"
	"time"
)

const (
	StateClosed   = "closed"
	StateOpen     = "open"
	StateHalfOpen = "half-open"
)

type Breaker struct {
	mu          sync.Mutex
	state       string
	failures    int
	successes   int
	threshold   int
	timeout     time.Duration
	lastFailure time.Time
	halfOpenMax int
	clock       func() time.Time
}

func NewBreaker(threshold int, timeout time.Duration) *Breaker {
	return &Breaker{
		state:       StateClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 1,
		clock:       time.Now,
	}
}

func NewBreakerWithClock(threshold int, timeout time.Duration, clock func() time.Time) *Breaker {
	return &Breaker{
		state:       StateClosed,
		threshold:   threshold,
		timeout:     timeout,
		halfOpenMax: 1,
		clock:       clock,
	}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		now := b.clock()
		if now.Sub(b.lastFailure) >= b.timeout {
			b.state = StateHalfOpen
			b.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return b.successes < b.halfOpenMax
	default:
		return false
	}
}

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

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures++
	b.lastFailure = b.clock()

	switch b.state {
	case StateClosed:
		if b.failures >= b.threshold {
			b.state = StateOpen
		}
	case StateHalfOpen:
		b.state = StateOpen
		b.successes = 0
	}
}

func (b *Breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

func (b *Breaker) Failures() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.failures
}

func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateClosed
	b.failures = 0
	b.successes = 0
	b.lastFailure = time.Time{}
}
