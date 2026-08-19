// Package store provides a persistent rate limiter that checkpoints its state
// to disk and recovers on restart.
//
// The Store wraps either a token bucket or a fixed/sliding window limiter.
// Every Allow/TryTake call modifies in-memory state; Checkpoint() persists the
// current state atomically. On Open, the store loads the last checkpoint if
// valid, or starts fresh if the file is missing or corrupt.
//
// Clock-rewind safety: if the caller-supplied timestamp is earlier than the
// last recorded time, the store refuses to grant tokens/allowances that would
// violate the configured rate. The token bucket clamps tokens to the value at
// the earlier timestamp without refilling; the fixed window rejects the request
// if already at capacity.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"rate-limit/internal/persist"
	"rate-limit/internal/tokenbucket"
	"rate-limit/internal/window"
)

const snapshotFile = "limiter.snap"

var (
	ErrClosed    = errors.New("store: closed")
	ErrNoAlgo    = errors.New("store: no algorithm configured")
	ErrBadConfig = errors.New("store: invalid configuration")
)

// Config describes the desired limiter algorithm and parameters.
type Config struct {
	Algo   string  // "bucket", "fixed", or "slide"
	Rate   float64 // tokens per second (bucket) or requests/window
	Burst  int     // max burst capacity / window limit
	Window time.Duration // window duration (fixed/slide); ignored for bucket
}

// Store is a persistent rate limiter.
type Store struct {
	dir      string
	snapPath string
	cfg      Config
	closed   bool

	// exactly one of these is non-nil after Open
	bucket *tokenbucket.Bucket
	fixed  *window.Fixed
	slide  *window.SlidingLog

	// stats
	allowed int
	denied  int

	// WasRecovered indicates whether state was loaded from a snapshot.
	WasRecovered bool
}

// Open opens or creates a persistent rate limiter in dir.
// If a valid snapshot exists and matches the configured algo, the state is
// restored. Otherwise a fresh limiter is created with now as the start time.
func Open(dir string, cfg Config, now time.Time) (*Store, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("store: mkdir: %w", err)
	}

	snapPath := filepath.Join(dir, snapshotFile)
	s := &Store{
		dir:      dir,
		snapPath: snapPath,
		cfg:      cfg,
	}

	// try to load snapshot
	snap, err := persist.Load(snapPath)
	if err == nil && algoMatch(snap.Algo, cfg.Algo) {
		if restored := s.restore(snap, now); restored {
			s.WasRecovered = true
			return s, nil
		}
	}

	// fresh start
	if err := s.createFresh(now); err != nil {
		return nil, err
	}
	return s, nil
}

// Allow checks if one request is allowed at time now.
func (s *Store) Allow(now time.Time) bool {
	return s.TryTake(now, 1)
}

// TryTake attempts to consume n tokens/requests at time now.
func (s *Store) TryTake(now time.Time, n int) bool {
	if s.closed {
		return false
	}
	var ok bool
	switch {
	case s.bucket != nil:
		ok = s.bucket.TryTake(now, n)
	case s.fixed != nil:
		ok = true
		for i := 0; i < n; i++ {
			if !s.fixed.Allow(now) {
				ok = false
			}
		}
		if n <= 0 {
			ok = false
		}
	case s.slide != nil:
		ok = true
		for i := 0; i < n; i++ {
			if !s.slide.Allow(now) {
				ok = false
			}
		}
		if n <= 0 {
			ok = false
		}
	}
	if ok {
		s.allowed++
	} else {
		s.denied++
	}
	return ok
}

// Tokens returns the current available tokens (bucket only, refilled to now).
func (s *Store) Tokens(now time.Time) float64 {
	if s.bucket != nil {
		return s.bucket.Tokens(now)
	}
	return 0
}

// Count returns the current window count (fixed window only).
func (s *Store) Count(now time.Time) int {
	if s.fixed != nil {
		return s.fixed.Count(now)
	}
	return 0
}

// Len returns the number of in-window stamps (sliding log only).
func (s *Store) Len(now time.Time) int {
	if s.slide != nil {
		return s.slide.Len(now)
	}
	return 0
}

// Stats returns (allowed, denied) counts since Open.
func (s *Store) Stats() (allowed, denied int) {
	return s.allowed, s.denied
}

// Checkpoint persists the current state to disk atomically.
func (s *Store) Checkpoint(now time.Time) error {
	if s.closed {
		return ErrClosed
	}
	snap := s.buildSnapshot(now)
	return persist.Save(s.snapPath, snap)
}

// Close marks the store as closed. It does NOT auto-checkpoint.
func (s *Store) Close() error {
	s.closed = true
	return nil
}

// Algo returns the active algorithm name.
func (s *Store) Algo() string { return s.cfg.Algo }

func (s *Store) buildSnapshot(now time.Time) *persist.Snapshot {
	switch {
	case s.bucket != nil:
		state := s.bucket.State()
		return &persist.Snapshot{Algo: persist.AlgoBucket, Bucket: &state}
	case s.fixed != nil:
		state := s.fixed.State()
		return &persist.Snapshot{Algo: persist.AlgoFixed, Fixed: &state}
	case s.slide != nil:
		state := s.slide.State(now)
		return &persist.Snapshot{Algo: persist.AlgoSlide, Slide: &state}
	default:
		return &persist.Snapshot{}
	}
}

func (s *Store) restore(snap *persist.Snapshot, now time.Time) bool {
	switch snap.Algo {
	case persist.AlgoBucket:
		if snap.Bucket == nil {
			return false
		}
		b, err := tokenbucket.RestoreBucket(*snap.Bucket)
		if err != nil {
			return false
		}
		s.bucket = b
		return true
	case persist.AlgoFixed:
		if snap.Fixed == nil {
			return false
		}
		f, err := window.RestoreFixed(*snap.Fixed)
		if err != nil {
			return false
		}
		s.fixed = f
		return true
	case persist.AlgoSlide:
		if snap.Slide == nil {
			return false
		}
		sl, err := window.RestoreSlidingLog(*snap.Slide)
		if err != nil {
			return false
		}
		s.slide = sl
		return true
	}
	return false
}

func (s *Store) createFresh(now time.Time) error {
	switch s.cfg.Algo {
	case "bucket":
		b, err := tokenbucket.NewBucket(s.cfg.Rate, s.cfg.Burst, now)
		if err != nil {
			return fmt.Errorf("store: new bucket: %w", err)
		}
		s.bucket = b
	case "fixed":
		win := s.cfg.Window
		if win <= 0 {
			win = time.Duration(float64(time.Second) * float64(s.cfg.Burst) / s.cfg.Rate)
		}
		f, err := window.NewFixed(s.cfg.Burst, win, now)
		if err != nil {
			return fmt.Errorf("store: new fixed: %w", err)
		}
		s.fixed = f
	case "slide":
		win := s.cfg.Window
		if win <= 0 {
			win = time.Duration(float64(time.Second) * float64(s.cfg.Burst) / s.cfg.Rate)
		}
		sl, err := window.NewSlidingLog(s.cfg.Burst, win)
		if err != nil {
			return fmt.Errorf("store: new slide: %w", err)
		}
		s.slide = sl
	default:
		return fmt.Errorf("%w: unknown algo %q", ErrNoAlgo, s.cfg.Algo)
	}
	return nil
}

func validateConfig(cfg Config) error {
	if cfg.Algo != "bucket" && cfg.Algo != "fixed" && cfg.Algo != "slide" {
		return fmt.Errorf("%w: algo must be bucket, fixed, or slide", ErrBadConfig)
	}
	if cfg.Rate <= 0 {
		return fmt.Errorf("%w: rate must be positive", ErrBadConfig)
	}
	if cfg.Burst < 1 {
		return fmt.Errorf("%w: burst must be at least 1", ErrBadConfig)
	}
	return nil
}

func algoMatch(code byte, name string) bool {
	switch code {
	case persist.AlgoBucket:
		return name == "bucket"
	case persist.AlgoFixed:
		return name == "fixed"
	case persist.AlgoSlide:
		return name == "slide"
	}
	return false
}
