package adaptive

import (
	"testing"
	"time"
)

func TestAdaptiveAllow(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	l := NewAdaptiveWithClock(5, 1, 20, clock)

	allowed := 0
	for i := 0; i < 10; i++ {
		if l.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("allowed = %d, want 5", allowed)
	}

	now = now.Add(time.Second)
	if !l.Allow() {
		t.Error("should allow after refill")
	}
}

func TestAdaptiveRateAdjustment(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	l := NewAdaptiveWithClock(10, 1, 20, clock)
	l.SetAdjustInterval(10)

	initialRate := l.CurrentRate()

	for i := 0; i < 8; i++ {
		l.RecordError()
	}
	for i := 0; i < 2; i++ {
		l.RecordSuccess()
	}

	newRate := l.CurrentRate()
	if newRate >= initialRate {
		t.Errorf("rate after errors = %v, should be less than %v", newRate, initialRate)
	}
}

func TestAdaptiveRateIncrease(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	l := NewAdaptiveWithClock(5, 1, 20, clock)
	l.SetAdjustInterval(10)

	for i := 0; i < 10; i++ {
		l.RecordError()
	}
	lowRate := l.CurrentRate()

	for i := 0; i < 10; i++ {
		l.RecordSuccess()
	}

	higherRate := l.CurrentRate()
	if higherRate <= lowRate {
		t.Errorf("rate after successes = %v, should be greater than %v", higherRate, lowRate)
	}
}

func TestAdaptiveRateBounds(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	l := NewAdaptiveWithClock(10, 2, 15, clock)
	l.SetAdjustInterval(5)

	for round := 0; round < 10; round++ {
		for i := 0; i < 5; i++ {
			l.RecordError()
		}
	}

	if rate := l.CurrentRate(); rate < 2 {
		t.Errorf("rate = %v, should not be below minRate 2", rate)
	}
}
