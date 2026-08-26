package gcra

import (
	"testing"
	"time"
)

func TestGCRAAllow(t *testing.T) {
	g := NewGCRA(10, 5)
	now := time.Now()

	if !g.Allow(now) {
		t.Error("first request should be allowed")
	}

	allowed := 1
	for i := 0; i < 10; i++ {
		if g.Allow(now) {
			allowed++
		}
	}
	if allowed > 6 {
		t.Errorf("allowed %d requests at once, expected <= 6 (burst+1)", allowed)
	}

	later := now.Add(time.Second)
	if !g.Allow(later) {
		t.Error("should allow after time passes")
	}
}

func TestGCRAAllowN(t *testing.T) {
	g := NewGCRA(10, 5)
	now := time.Now()

	if !g.AllowN(now, 0) {
		t.Error("AllowN(0) should always return true")
	}

	if !g.AllowN(now, -1) {
		t.Error("AllowN(-1) should always return true")
	}

	g.Reset()
	if !g.AllowN(now, 3) {
		t.Error("AllowN(3) should succeed with burst=5")
	}

	g.Reset()
	if g.AllowN(now, 100) {
		t.Error("AllowN(100) should fail with burst=5")
	}
}

func TestGCRARateAndBurst(t *testing.T) {
	g := NewGCRA(5, 3)

	if r := g.Rate(); r != 5 {
		t.Errorf("Rate() = %v, want 5", r)
	}
	if b := g.Burst(); b != 3 {
		t.Errorf("Burst() = %d, want 3", b)
	}

	expectedEmission := time.Duration(float64(time.Second) / 5)
	if e := g.Emission(); e != expectedEmission {
		t.Errorf("Emission() = %v, want %v", e, expectedEmission)
	}
}

func TestGCRAReset(t *testing.T) {
	g := NewGCRA(10, 5)
	now := time.Now()

	for i := 0; i < 10; i++ {
		g.Allow(now)
	}

	g.Reset()

	if !g.Allow(now) {
		t.Error("should allow after reset")
	}

	if tat := g.TAT(); tat.IsZero() {
		t.Error("TAT should not be zero after Allow")
	}
}
