package priority

import (
	"testing"
	"time"
)

func TestPriorityAllow(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	p := NewPriorityWithClock(3, 2, clock)

	allowed := 0
	for i := 0; i < 10; i++ {
		if p.Allow(0) {
			allowed++
		}
	}
	if allowed != 6 {
		t.Errorf("priority 0 allowed = %d, want 6", allowed)
	}

	allowed = 0
	for i := 0; i < 10; i++ {
		if p.Allow(2) {
			allowed++
		}
	}
	if allowed != 2 {
		t.Errorf("priority 2 allowed = %d, want 2", allowed)
	}
}

func TestPriorityInvalidLevel(t *testing.T) {
	p := NewPriority(3, 10)

	if p.Allow(-1) {
		t.Error("Allow(-1) should return false")
	}
	if p.Allow(3) {
		t.Error("Allow(3) should return false for 3-level limiter")
	}
	if p.Allow(100) {
		t.Error("Allow(100) should return false")
	}
}

func TestPrioritySetRate(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	p := NewPriorityWithClock(2, 5, clock)

	p.SetRate(1, 20)

	if rate := p.Rate(1); rate != 20 {
		t.Errorf("rate after SetRate = %v, want 20", rate)
	}

	p.SetRate(-1, 100)
	p.SetRate(5, 100)
}

func TestPriorityRefill(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	p := NewPriorityWithClock(2, 5, clock)

	for p.Allow(0) {
	}

	now = now.Add(time.Second)
	if !p.Allow(0) {
		t.Error("should allow after refill")
	}
}
