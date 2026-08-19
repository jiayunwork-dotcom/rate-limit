package window

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)

func TestNewRejectsBadParams(t *testing.T) {
	if _, err := NewFixed(0, time.Minute, t0); err == nil {
		t.Error("fixed: expected error for zero limit")
	}
	if _, err := NewFixed(2, 0, t0); err == nil {
		t.Error("fixed: expected error for zero window")
	}
	if _, err := NewSlidingLog(0, time.Minute); err == nil {
		t.Error("sliding: expected error for zero limit")
	}
	if _, err := NewSlidingLog(2, -time.Second); err == nil {
		t.Error("sliding: expected error for negative window")
	}
}

func TestFixedWindowResets(t *testing.T) {
	w, _ := NewFixed(2, time.Minute, t0)
	if !w.Allow(t0) || !w.Allow(t0.Add(time.Second)) {
		t.Fatal("first two requests should pass")
	}
	if w.Allow(t0.Add(2 * time.Second)) {
		t.Error("third request in same window should be denied")
	}
	if got := w.Count(t0); got != 2 {
		t.Errorf("Count = %d, want 2", got)
	}
	if !w.Allow(t0.Add(time.Minute)) {
		t.Error("request in next window should pass")
	}
	if got := w.Count(t0.Add(time.Minute)); got != 1 {
		t.Errorf("Count after reset = %d, want 1", got)
	}
}

func TestFixedWindowSkipsIdleWindows(t *testing.T) {
	w, _ := NewFixed(2, time.Minute, t0)
	w.Allow(t0)
	if !w.Allow(t0.Add(10 * time.Minute)) {
		t.Error("long idle gap must reset the window")
	}
}

func TestSlidingLogEvictsOld(t *testing.T) {
	w, _ := NewSlidingLog(2, time.Minute)
	if !w.Allow(t0) || !w.Allow(t0.Add(10*time.Second)) {
		t.Fatal("first two should pass")
	}
	if w.Allow(t0.Add(20 * time.Second)) {
		t.Error("third inside window should be denied")
	}
	if !w.Allow(t0.Add(time.Minute)) {
		t.Error("first stamp should have been evicted by now")
	}
	if got := w.Len(t0.Add(time.Minute)); got != 2 {
		t.Errorf("Len = %d, want 2", got)
	}
}

func TestSlidingLogBoundary(t *testing.T) {
	w, _ := NewSlidingLog(1, time.Minute)
	if !w.Allow(t0) {
		t.Fatal("first should pass")
	}
	if w.Allow(t0.Add(time.Minute - time.Nanosecond)) {
		t.Error("still inside window")
	}
	if !w.Allow(t0.Add(time.Minute)) {
		t.Error("exactly one window later the old entry is evicted")
	}
}

func TestFixedStateAndRestore(t *testing.T) {
	w, _ := NewFixed(5, 30*time.Second, t0)
	w.Allow(t0)
	w.Allow(t0.Add(1 * time.Second))
	w.Allow(t0.Add(2 * time.Second))

	state := w.State()
	if state.Limit != 5 {
		t.Errorf("Limit = %d, want 5", state.Limit)
	}
	if state.Count != 3 {
		t.Errorf("Count = %d, want 3", state.Count)
	}
	if state.Window != 30*time.Second {
		t.Errorf("Window = %v, want 30s", state.Window)
	}

	restored, err := RestoreFixed(state)
	if err != nil {
		t.Fatalf("RestoreFixed: %v", err)
	}
	if restored.Count(t0.Add(3*time.Second)) != 3 {
		t.Error("restored count should be 3")
	}
	// 2 more allowed
	if !restored.Allow(t0.Add(3 * time.Second)) {
		t.Error("4th should pass")
	}
	if !restored.Allow(t0.Add(4 * time.Second)) {
		t.Error("5th should pass")
	}
	if restored.Allow(t0.Add(5 * time.Second)) {
		t.Error("6th should be denied (limit=5)")
	}
}

func TestFixedRestoreRejectsBadState(t *testing.T) {
	cases := []struct {
		name  string
		state FixedState
	}{
		{"zero limit", FixedState{Limit: 0, Window: time.Second, Count: 0, Start: t0}},
		{"zero window", FixedState{Limit: 5, Window: 0, Count: 0, Start: t0}},
		{"count > limit", FixedState{Limit: 5, Window: time.Second, Count: 10, Start: t0}},
		{"zero start", FixedState{Limit: 5, Window: time.Second, Count: 0}},
	}
	for _, tc := range cases {
		if _, err := RestoreFixed(tc.state); err == nil {
			t.Errorf("RestoreFixed(%s) should error", tc.name)
		}
	}
}

func TestSlidingLogStateAndRestore(t *testing.T) {
	w, _ := NewSlidingLog(5, 30*time.Second)
	w.Allow(t0.Add(1 * time.Second))
	w.Allow(t0.Add(2 * time.Second))

	state := w.State(t0.Add(3 * time.Second))
	if state.Limit != 5 {
		t.Errorf("Limit = %d, want 5", state.Limit)
	}
	if len(state.Stamps) != 2 {
		t.Fatalf("Stamps len = %d, want 2", len(state.Stamps))
	}

	restored, err := RestoreSlidingLog(state)
	if err != nil {
		t.Fatalf("RestoreSlidingLog: %v", err)
	}
	if restored.Len(t0.Add(3*time.Second)) != 2 {
		t.Error("restored len should be 2")
	}
	// 3 more allowed
	for i := 0; i < 3; i++ {
		if !restored.Allow(t0.Add(time.Duration(4+i) * time.Second)) {
			t.Errorf("Allow #%d should pass", i+1)
		}
	}
	if restored.Allow(t0.Add(7 * time.Second)) {
		t.Error("6th should be denied (limit=5)")
	}
}

func TestSlidingLogRestoreRejectsBadState(t *testing.T) {
	if _, err := RestoreSlidingLog(SlidingLogState{Limit: 0, Window: time.Second}); err == nil {
		t.Error("expected error for zero limit")
	}
	if _, err := RestoreSlidingLog(SlidingLogState{Limit: 5, Window: 0}); err == nil {
		t.Error("expected error for zero window")
	}
}
