package leaky

import (
	"testing"
	"time"
)

func TestBucketAllow(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewWithClock(10, 5, clock)

	for i := 0; i < 5; i++ {
		if !b.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if b.Allow() {
		t.Error("request should be denied when bucket is full")
	}

	now = now.Add(time.Second)
	if !b.Allow() {
		t.Error("request should be allowed after time passes")
	}
}

func TestBucketAllowN(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewWithClock(5, 10, clock)

	if !b.AllowN(8) {
		t.Error("AllowN(8) should succeed with capacity 10")
	}

	if b.AllowN(5) {
		t.Error("AllowN(5) should fail, only 2 capacity remaining")
	}

	if !b.AllowN(0) {
		t.Error("AllowN(0) should always return true")
	}
	if !b.AllowN(-1) {
		t.Error("AllowN(-1) should always return true")
	}
}

func TestBucketWait(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewWithClock(10, 3, clock)

	if wait := b.Wait(); wait != 0 {
		t.Errorf("empty bucket wait = %v, want 0", wait)
	}

	b.AllowN(3)

	if wait := b.Wait(); wait <= 0 {
		t.Errorf("full bucket wait = %v, want > 0", wait)
	}
}

func TestBucketWaterLevel(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	b := NewWithClock(10, 5, clock)

	if level := b.WaterLevel(); level != 0 {
		t.Errorf("initial water level = %v, want 0", level)
	}

	b.AllowN(3)
	if level := b.WaterLevel(); level != 3 {
		t.Errorf("water level after AllowN(3) = %v, want 3", level)
	}
}
