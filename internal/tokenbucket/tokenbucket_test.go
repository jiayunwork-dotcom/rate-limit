package tokenbucket

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)

func TestNewBucketRejectsBadParams(t *testing.T) {
	if _, err := NewBucket(0, 5, t0); err == nil {
		t.Error("expected error for zero rate")
	}
	if _, err := NewBucket(-1, 5, t0); err == nil {
		t.Error("expected error for negative rate")
	}
	if _, err := NewBucket(2, 0, t0); err == nil {
		t.Error("expected error for zero burst")
	}
	if _, err := NewBucket(2, -3, t0); err == nil {
		t.Error("expected error for negative burst")
	}
}

func TestBurstCapacity(t *testing.T) {
	b, err := NewBucket(1, 3, t0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if !b.Allow(t0) {
			t.Fatalf("request %d should pass within burst", i+1)
		}
	}
	if b.Allow(t0) {
		t.Error("4th request at same instant should be denied")
	}
}

func TestRefillOverTime(t *testing.T) {
	b, _ := NewBucket(1, 2, t0) // 1 token/sec
	b.Allow(t0)                 // tokens: 1
	b.Allow(t0.Add(500 * time.Millisecond))
	// tokens: 1 -> 1.5 refill, then 0.5 after taking one.
	if b.Allow(t0.Add(900 * time.Millisecond)) {
		t.Error("request with only 0.9 tokens should be denied")
	}
	if !b.Allow(t0.Add(1500 * time.Millisecond)) {
		t.Error("request after full refill window should pass")
	}
	if b.Allow(t0.Add(1600 * time.Millisecond)) {
		t.Error("only 0.6 tokens refilled since, should be denied")
	}
}

func TestTryTakeBatch(t *testing.T) {
	b, _ := NewBucket(1, 5, t0)
	if !b.TryTake(t0, 5) {
		t.Error("batch of 5 should pass")
	}
	if b.TryTake(t0, 1) {
		t.Error("bucket is empty")
	}
	if b.TryTake(t0, 0) {
		t.Error("n=0 must be rejected")
	}
	if b.TryTake(t0, 6) {
		t.Error("n greater than burst must be rejected")
	}
}

func TestTokensCapAndSetRate(t *testing.T) {
	b, _ := NewBucket(1, 4, t0)
	if got := b.Tokens(t0.Add(time.Hour)); got != 4 {
		t.Errorf("tokens should cap at burst, got %v", got)
	}
	if err := b.SetRate(0); err == nil {
		t.Error("SetRate(0) should fail")
	}
	if err := b.SetRate(5); err != nil {
		t.Fatalf("SetRate(5): %v", err)
	}
	if got := b.Tokens(t0.Add(time.Hour + time.Second)); got != 4 {
		t.Errorf("capped tokens = %v, want 4", got)
	}
}

func TestStateAndRestore(t *testing.T) {
	b, _ := NewBucket(2.5, 10, t0)
	b.TryTake(t0, 4)

	state := b.State()
	if state.Rate != 2.5 {
		t.Errorf("State.Rate = %f, want 2.5", state.Rate)
	}
	if state.Burst != 10 {
		t.Errorf("State.Burst = %f, want 10", state.Burst)
	}
	if state.Tokens != 6 {
		t.Errorf("State.Tokens = %f, want 6", state.Tokens)
	}
	if !state.Last.Equal(t0) {
		t.Errorf("State.Last = %v, want %v", state.Last, t0)
	}

	restored, err := RestoreBucket(state)
	if err != nil {
		t.Fatalf("RestoreBucket: %v", err)
	}
	if restored.Tokens(t0) != 6 {
		t.Errorf("restored tokens = %f, want 6", restored.Tokens(t0))
	}
	if restored.Rate() != 2.5 {
		t.Errorf("restored rate = %f, want 2.5", restored.Rate())
	}
}

func TestRestoreBucketRejectsBadState(t *testing.T) {
	cases := []struct {
		name  string
		state BucketState
	}{
		{"zero rate", BucketState{Rate: 0, Burst: 5, Tokens: 3, Last: t0}},
		{"negative burst", BucketState{Rate: 1, Burst: 0, Tokens: 0, Last: t0}},
		{"tokens exceed burst", BucketState{Rate: 1, Burst: 5, Tokens: 10, Last: t0}},
		{"zero time", BucketState{Rate: 1, Burst: 5, Tokens: 3}},
	}
	for _, tc := range cases {
		if _, err := RestoreBucket(tc.state); err == nil {
			t.Errorf("RestoreBucket(%s) should error", tc.name)
		}
	}
}

func TestClockRewindNoExtraTokens(t *testing.T) {
	b, _ := NewBucket(1, 5, t0)
	b.TryTake(t0, 5)

	// advance 3s -> refill 3
	later := t0.Add(3 * time.Second)
	if got := b.Tokens(later); got < 2.99 || got > 3.01 {
		t.Fatalf("tokens at later = %f, want ~3", got)
	}

	// rewind clock: call Tokens with earlier time
	rewind := t0.Add(1 * time.Second)
	toks := b.Tokens(rewind)
	// refill must NOT rewind; tokens should stay at 3 (last was "later")
	if toks > 3.01 {
		t.Errorf("tokens after rewind = %f, should not exceed 3", toks)
	}
}
