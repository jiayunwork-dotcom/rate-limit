package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreOpenFreshAllow(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	cfg := Config{Algo: "bucket", Rate: 2, Burst: 5}
	s, err := Open(dir, cfg, now)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if s.WasRecovered {
		t.Error("fresh store should not be recovered")
	}

	// burst allows 5 immediately
	for i := 0; i < 5; i++ {
		if !s.Allow(now) {
			t.Errorf("Allow #%d should succeed (within burst)", i+1)
		}
	}
	// 6th should fail (no time for refill)
	if s.Allow(now) {
		t.Error("Allow #6 should fail (burst exhausted)")
	}

	allowed, denied := s.Stats()
	if allowed != 5 {
		t.Errorf("allowed = %d, want 5", allowed)
	}
	if denied != 1 {
		t.Errorf("denied = %d, want 1", denied)
	}
}

func TestStoreCheckpointAndRecovery(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 2, Burst: 10}

	// session 1: consume tokens, checkpoint
	s, _ := Open(dir, cfg, now)
	for i := 0; i < 6; i++ {
		s.Allow(now)
	}
	// tokens should be 4 (10 - 6)
	toksBefore := s.Tokens(now)
	if err := s.Checkpoint(now); err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	s.Close()

	// session 2: reopen, token count should match
	s2, _ := Open(dir, cfg, now)
	defer s2.Close()

	if !s2.WasRecovered {
		t.Error("expected WasRecovered=true")
	}
	toksAfter := s2.Tokens(now)
	if diff := toksBefore - toksAfter; diff < -0.01 || diff > 0.01 {
		t.Errorf("tokens after recovery = %.2f, want %.2f", toksAfter, toksBefore)
	}
}

func TestStoreRecoveryBucketTokens(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 1, Burst: 5}

	s, _ := Open(dir, cfg, now)
	// consume 3 tokens
	s.TryTake(now, 3)
	// advance 1 second (refills 1 token) -> should have 3
	later := now.Add(1 * time.Second)
	toks := s.Tokens(later)
	s.Checkpoint(later)
	s.Close()

	// recover
	s2, _ := Open(dir, cfg, later)
	defer s2.Close()

	recovered := s2.Tokens(later)
	if diff := toks - recovered; diff < -0.01 || diff > 0.01 {
		t.Errorf("recovered tokens = %.2f, want %.2f", recovered, toks)
	}
}

func TestStoreFixedWindowRecoveryCountPreserved(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "fixed", Rate: 1, Burst: 10, Window: 60 * time.Second}

	// session 1: use 7 of 10 allowance
	s, _ := Open(dir, cfg, now)
	for i := 0; i < 7; i++ {
		s.Allow(now)
	}
	countBefore := s.Count(now)
	s.Checkpoint(now)
	s.Close()

	// session 2: reopen within same window
	s2, _ := Open(dir, cfg, now.Add(5*time.Second))
	defer s2.Close()

	if !s2.WasRecovered {
		t.Fatal("expected recovery")
	}
	countAfter := s2.Count(now.Add(5 * time.Second))
	if countAfter != countBefore {
		t.Errorf("count after recovery = %d, want %d", countAfter, countBefore)
	}

	// only 3 more should be allowed
	for i := 0; i < 3; i++ {
		if !s2.Allow(now.Add(5 * time.Second)) {
			t.Errorf("Allow #%d should succeed (3 remaining)", i+1)
		}
	}
	if s2.Allow(now.Add(5 * time.Second)) {
		t.Error("should deny after window limit reached")
	}
}

func TestStoreClockRewindSafety(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 1, Burst: 5}

	s, _ := Open(dir, cfg, now)
	// consume all 5 tokens
	s.TryTake(now, 5)

	// advance 3 seconds -> refill 3 tokens
	later := now.Add(3 * time.Second)
	toksAtLater := s.Tokens(later)
	if toksAtLater < 2.99 || toksAtLater > 3.01 {
		t.Fatalf("tokens at later = %.2f, want ~3.0", toksAtLater)
	}

	// checkpoint at the later time
	s.Checkpoint(later)
	s.Close()

	// simulate clock rewind: reopen with earlier timestamp
	rewind := now.Add(1 * time.Second) // 2 seconds before checkpoint
	s2, _ := Open(dir, cfg, rewind)
	defer s2.Close()

	// the bucket was checkpointed with last=later, tokens=3
	// clock rewind means now < last; refill must NOT add tokens
	// tokens should remain at most what was checkpointed (3), not more
	toksAfterRewind := s2.Tokens(rewind)
	if toksAfterRewind > 3.01 {
		t.Errorf("tokens after clock rewind = %.2f, should not exceed checkpoint value 3.0", toksAfterRewind)
	}

	// consume the 3 tokens
	if !s2.TryTake(rewind, 3) {
		t.Error("should allow taking 3 tokens")
	}
	// now should be empty
	if s2.Allow(rewind) {
		t.Error("should deny after exhausting tokens (clock rewind must not grant extra)")
	}
}

func TestStoreCorruptSnapshotStartsFresh(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, snapshotFile)

	// write garbage
	os.WriteFile(snapPath, []byte("corrupt data here!!!"), 0644)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 2, Burst: 5}

	s, err := Open(dir, cfg, now)
	if err != nil {
		t.Fatalf("Open with corrupt snapshot should not error: %v", err)
	}
	defer s.Close()

	if s.WasRecovered {
		t.Error("should not be recovered from corrupt snapshot")
	}
	// should behave as fresh (full burst)
	if !s.TryTake(now, 5) {
		t.Error("fresh store should allow burst")
	}
}

func TestStoreTruncatedSnapshotRejects(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 2, Burst: 5}

	// create valid snapshot
	s, _ := Open(dir, cfg, now)
	s.TryTake(now, 3)
	s.Checkpoint(now)
	s.Close()

	// truncate the file
	snapPath := filepath.Join(dir, snapshotFile)
	data, _ := os.ReadFile(snapPath)
	os.WriteFile(snapPath, data[:len(data)/2], 0644)

	// reopen should fall back to fresh
	s2, err := Open(dir, cfg, now)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s2.Close()

	if s2.WasRecovered {
		t.Error("should not recover from truncated snapshot")
	}
	// fresh: full burst available
	if !s2.TryTake(now, 5) {
		t.Error("fresh store should have full burst")
	}
}

func TestStoreWindowBoundaryTick(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	winDur := 10 * time.Second
	cfg := Config{Algo: "fixed", Rate: 1, Burst: 3, Window: winDur}

	s, _ := Open(dir, cfg, start)
	defer s.Close()

	// fill to limit
	s.Allow(start)
	s.Allow(start.Add(2 * time.Second))
	s.Allow(start.Add(4 * time.Second))

	// at window boundary exactly, deny
	atBoundary := start.Add(winDur - 1*time.Nanosecond)
	if s.Allow(atBoundary) {
		t.Error("should deny at boundary (window not yet reset, limit reached)")
	}

	// at exactly window start + winDur, window resets
	afterReset := start.Add(winDur)
	if !s.Allow(afterReset) {
		t.Error("should allow after window reset")
	}

	// checkpoint and recover at boundary
	s.Checkpoint(afterReset)
	s.Close()

	s2, _ := Open(dir, cfg, afterReset)
	defer s2.Close()

	// should have count=1 (the one we just allowed after reset)
	if s2.Count(afterReset) != 1 {
		t.Errorf("count after recovery at boundary = %d, want 1", s2.Count(afterReset))
	}
}

func TestStoreSlidingLogRecovery(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "slide", Rate: 1, Burst: 5, Window: 10 * time.Second}

	s, _ := Open(dir, cfg, start)
	// add 3 requests
	s.Allow(start.Add(1 * time.Second))
	s.Allow(start.Add(2 * time.Second))
	s.Allow(start.Add(3 * time.Second))

	checkTime := start.Add(4 * time.Second)
	lenBefore := s.Len(checkTime)
	s.Checkpoint(checkTime)
	s.Close()

	// recover
	s2, _ := Open(dir, cfg, checkTime)
	defer s2.Close()

	if !s2.WasRecovered {
		t.Fatal("expected recovery")
	}
	lenAfter := s2.Len(checkTime)
	if lenAfter != lenBefore {
		t.Errorf("len after recovery = %d, want %d", lenAfter, lenBefore)
	}

	// should allow 2 more (5 - 3 = 2)
	if !s2.Allow(checkTime) {
		t.Error("should allow (2 remaining)")
	}
	if !s2.Allow(checkTime) {
		t.Error("should allow (1 remaining)")
	}
	if s2.Allow(checkTime) {
		t.Error("should deny (limit reached)")
	}
}

func TestStoreMultipleCheckpoints(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 1, Burst: 10}

	s, _ := Open(dir, cfg, now)
	s.TryTake(now, 3)
	s.Checkpoint(now)
	s.TryTake(now, 2)
	later := now.Add(5 * time.Second)
	s.Checkpoint(later) // second checkpoint replaces first
	s.Close()

	s2, _ := Open(dir, cfg, later)
	defer s2.Close()

	// started with 10, took 5, refilled 5 seconds * 1/s = 5 -> tokens = 10 (capped at burst)
	toks := s2.Tokens(later)
	if toks < 9.99 || toks > 10.01 {
		t.Errorf("tokens = %.2f, want 10 (refill capped at burst)", toks)
	}
}

func TestStoreClosedOperations(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	cfg := Config{Algo: "bucket", Rate: 1, Burst: 5}

	s, _ := Open(dir, cfg, now)
	s.Close()

	if s.Allow(now) {
		t.Error("Allow on closed store should return false")
	}
	if err := s.Checkpoint(now); err == nil {
		t.Error("Checkpoint on closed store should error")
	}
}

func TestStoreBadConfig(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	cases := []Config{
		{Algo: "unknown", Rate: 1, Burst: 5},
		{Algo: "bucket", Rate: -1, Burst: 5},
		{Algo: "bucket", Rate: 1, Burst: 0},
	}
	for _, cfg := range cases {
		_, err := Open(dir, cfg, now)
		if err == nil {
			t.Errorf("Open(%+v) should error", cfg)
		}
	}
}
