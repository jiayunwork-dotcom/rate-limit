package circuit

import (
	"testing"
	"time"
)

func TestBreakerClosedState(t *testing.T) {
	b := NewBreaker(3, time.Second)

	if s := b.State(); s != StateClosed {
		t.Errorf("initial state = %q, want %q", s, StateClosed)
	}

	if !b.Allow() {
		t.Error("closed breaker should allow requests")
	}

	b.RecordFailure()
	b.RecordFailure()
	if s := b.State(); s != StateClosed {
		t.Errorf("state after 2 failures = %q, want %q", s, StateClosed)
	}
}

func TestBreakerOpenState(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewBreakerWithClock(3, time.Second, clock)

	b.RecordFailure()
	b.RecordFailure()
	b.RecordFailure()

	if s := b.State(); s != StateOpen {
		t.Errorf("state after threshold failures = %q, want %q", s, StateOpen)
	}

	if b.Allow() {
		t.Error("open breaker should deny requests")
	}

	now = now.Add(2 * time.Second)
	if !b.Allow() {
		t.Error("breaker should allow after timeout (half-open)")
	}

	if s := b.State(); s != StateHalfOpen {
		t.Errorf("state after timeout = %q, want %q", s, StateHalfOpen)
	}
}

func TestBreakerHalfOpenRecovery(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewBreakerWithClock(2, time.Second, clock)

	b.RecordFailure()
	b.RecordFailure()

	now = now.Add(2 * time.Second)
	b.Allow()

	b.RecordSuccess()
	if s := b.State(); s != StateClosed {
		t.Errorf("state after success in half-open = %q, want %q", s, StateClosed)
	}
}

func TestBreakerHalfOpenFailure(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewBreakerWithClock(2, time.Second, clock)

	b.RecordFailure()
	b.RecordFailure()

	now = now.Add(2 * time.Second)
	b.Allow()

	b.RecordFailure()
	if s := b.State(); s != StateOpen {
		t.Errorf("state after failure in half-open = %q, want %q", s, StateOpen)
	}
}

func TestBreakerReset(t *testing.T) {
	b := NewBreaker(2, time.Second)
	b.RecordFailure()
	b.RecordFailure()

	b.Reset()
	if s := b.State(); s != StateClosed {
		t.Errorf("state after reset = %q, want %q", s, StateClosed)
	}
	if f := b.Failures(); f != 0 {
		t.Errorf("failures after reset = %d, want 0", f)
	}
}
