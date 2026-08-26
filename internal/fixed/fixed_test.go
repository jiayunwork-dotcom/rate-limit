package fixed

import (
	"testing"
	"time"
)

func TestCounterAllow(t *testing.T) {
	c := NewCounter(5, time.Second)
	now := time.Now()

	for i := 0; i < 5; i++ {
		if !c.Allow(now) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if c.Allow(now) {
		t.Error("request should be denied when limit reached")
	}

	next := now.Add(2 * time.Second)
	if !c.Allow(next) {
		t.Error("request should be allowed in new window")
	}
}

func TestCounterRemaining(t *testing.T) {
	c := NewCounter(10, time.Second)
	now := time.Now()

	c.Allow(now)
	remaining := c.Remaining()
	if remaining != 9 {
		t.Errorf("remaining after 1 request = %d, want 9", remaining)
	}

	for i := 0; i < 5; i++ {
		c.Allow(now)
	}
	remaining = c.Remaining()
	if remaining != 4 {
		t.Errorf("remaining after 6 requests = %d, want 4", remaining)
	}
}

func TestCounterReset(t *testing.T) {
	c := NewCounter(5, time.Second)
	now := time.Now()

	c.Allow(now)
	c.Allow(now)
	c.Reset()

	if r := c.Remaining(); r != 5 {
		t.Errorf("remaining after reset = %d, want 5", r)
	}
}
