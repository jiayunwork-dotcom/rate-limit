// Package window implements fixed window and sliding window log limiters.
package window

import (
	"fmt"
	"time"
)

// Fixed is a counter that resets at the start of every window. The first
// request defines the window origin.
type Fixed struct {
	limit  int
	window time.Duration
	count  int
	start  time.Time
}

// NewFixed creates a fixed window limiter of limit requests per window.
func NewFixed(limit int, window time.Duration, now time.Time) (*Fixed, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1, got %d", limit)
	}
	if window <= 0 {
		return nil, fmt.Errorf("window must be positive, got %v", window)
	}
	return &Fixed{limit: limit, window: window, start: now}, nil
}

func (w *Fixed) roll(now time.Time) {
	if now.Sub(w.start) >= w.window {
		// Skip whole windows that may have passed unused.
		w.start = now
		w.count = 0
	}
}

// Allow records one request if under the limit.
func (w *Fixed) Allow(now time.Time) bool {
	w.roll(now)
	if w.count >= w.limit {
		return false
	}
	w.count++
	return true
}

// Count reports requests recorded in the current window.
func (w *Fixed) Count(now time.Time) int {
	w.roll(now)
	return w.count
}

// SlidingLog keeps timestamps of accepted requests and evicts entries older
// than the window, allowing at most limit inside any window span.
type SlidingLog struct {
	limit  int
	window time.Duration
	stamps []time.Time
}

// NewSlidingLog creates a sliding window log limiter.
func NewSlidingLog(limit int, window time.Duration) (*SlidingLog, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1, got %d", limit)
	}
	if window <= 0 {
		return nil, fmt.Errorf("window must be positive, got %v", window)
	}
	return &SlidingLog{limit: limit, window: window}, nil
}

func (w *SlidingLog) evict(now time.Time) {
	keep := w.stamps[:0]
	for _, ts := range w.stamps {
		if now.Sub(ts) < w.window {
			keep = append(keep, ts)
		}
	}
	w.stamps = keep
}

// Allow records one request if fewer than limit requests fall inside the
// trailing window.
func (w *SlidingLog) Allow(now time.Time) bool {
	w.evict(now)
	if len(w.stamps) >= w.limit {
		return false
	}
	w.stamps = append(w.stamps, now)
	return true
}

// Len reports the number of in-window requests.
func (w *SlidingLog) Len(now time.Time) int {
	w.evict(now)
	return len(w.stamps)
}

// FixedState holds the serializable state of a fixed window limiter.
type FixedState struct {
	Limit  int
	Window time.Duration
	Count  int
	Start  time.Time
}

// State returns the current fixed window state for serialization.
func (w *Fixed) State() FixedState {
	return FixedState{
		Limit:  w.limit,
		Window: w.window,
		Count:  w.count,
		Start:  w.start,
	}
}

// RestoreFixed creates a Fixed window limiter from a previously saved state.
func RestoreFixed(s FixedState) (*Fixed, error) {
	if s.Limit < 1 {
		return nil, fmt.Errorf("invalid limit in state: %d", s.Limit)
	}
	if s.Window <= 0 {
		return nil, fmt.Errorf("invalid window in state: %v", s.Window)
	}
	if s.Count < 0 || s.Count > s.Limit {
		return nil, fmt.Errorf("count %d out of range [0, %d]", s.Count, s.Limit)
	}
	if s.Start.IsZero() {
		return nil, fmt.Errorf("start timestamp is zero")
	}
	return &Fixed{
		limit:  s.Limit,
		window: s.Window,
		count:  s.Count,
		start:  s.Start,
	}, nil
}

// Limit returns the configured request limit per window.
func (w *Fixed) Limit() int { return w.limit }

// WindowDuration returns the configured window duration.
func (w *Fixed) WindowDuration() time.Duration { return w.window }

// SlidingLogState holds the serializable state of a sliding log limiter.
type SlidingLogState struct {
	Limit  int
	Window time.Duration
	Stamps []time.Time
}

// State returns the current sliding log state for serialization.
func (w *SlidingLog) State(now time.Time) SlidingLogState {
	w.evict(now)
	stamps := make([]time.Time, len(w.stamps))
	copy(stamps, w.stamps)
	return SlidingLogState{
		Limit:  w.limit,
		Window: w.window,
		Stamps: stamps,
	}
}

// RestoreSlidingLog creates a SlidingLog from a previously saved state.
func RestoreSlidingLog(s SlidingLogState) (*SlidingLog, error) {
	if s.Limit < 1 {
		return nil, fmt.Errorf("invalid limit in state: %d", s.Limit)
	}
	if s.Window <= 0 {
		return nil, fmt.Errorf("invalid window in state: %v", s.Window)
	}
	stamps := make([]time.Time, len(s.Stamps))
	copy(stamps, s.Stamps)
	return &SlidingLog{
		limit:  s.Limit,
		window: s.Window,
		stamps: stamps,
	}, nil
}

// SlidingLimit returns the configured request limit.
func (w *SlidingLog) SlidingLimit() int { return w.limit }

// SlidingWindow returns the configured window duration.
func (w *SlidingLog) SlidingWindow() time.Duration { return w.window }
