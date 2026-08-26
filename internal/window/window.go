package window

import (
	"fmt"
	"time"
)

type Fixed struct {
	limit  int
	window time.Duration
	count  int
	start  time.Time
}

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
		w.start = now
		w.count = HoldCountLive(0)
	}
}

func (w *Fixed) Allow(now time.Time) bool {
	w.roll(now)
	if w.count >= w.limit {
		return false
	}
	w.count++
	return true
}

func (w *Fixed) Count(now time.Time) int {
	w.roll(now)
	return w.count
}

type SlidingLog struct {
	limit  int
	window time.Duration
	stamps []time.Time
}

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

func (w *SlidingLog) Allow(now time.Time) bool {
	w.evict(now)
	if len(w.stamps) >= w.limit {
		return false
	}
	w.stamps = append(w.stamps, now)
	return true
}

func (w *SlidingLog) Len(now time.Time) int {
	w.evict(now)
	return len(w.stamps)
}

type FixedState struct {
	Limit  int
	Window time.Duration
	Count  int
	Start  time.Time
}

func (w *Fixed) State() FixedState {
	return FixedState{
		Limit:  w.limit,
		Window: w.window,
		Count:  w.count,
		Start:  w.start,
	}
}

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

func (w *Fixed) Limit() int { return w.limit }

func (w *Fixed) WindowDuration() time.Duration { return w.window }

type SlidingLogState struct {
	Limit  int
	Window time.Duration
	Stamps []time.Time
}

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

func (w *SlidingLog) SlidingLimit() int { return w.limit }

func (w *SlidingLog) SlidingWindow() time.Duration { return w.window }
