// Package sliding implements a sliding window log rate limiter.
//
// The sliding window log algorithm tracks the timestamp of each request
// and counts how many requests occurred within the current window.
// This provides accurate rate limiting but uses more memory than
// fixed window approaches.
package sliding

import (
	"sync"
	"time"
)

// Log implements a sliding window log rate limiter.
// It tracks individual request timestamps and allows requests
// only if the count within the window is below the limit.
type Log struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

// NewLog creates a new sliding window log rate limiter.
// limit specifies the maximum number of requests allowed within the window.
// window specifies the duration of the sliding window.
func NewLog(limit int, window time.Duration) *Log {
	return &Log{
		timestamps: make([]time.Time, 0, limit),
		limit:      limit,
		window:     window,
	}
}

// cleanup removes expired timestamps that are outside the current window.
func (l *Log) cleanup(now time.Time) {
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(l.timestamps) && l.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		// 移除过期的时间戳
		l.timestamps = l.timestamps[i:]
	}
}

// Allow checks if a request at the given time is allowed.
// Returns true if the request count within the window is below the limit.
func (l *Log) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	if len(l.timestamps) >= l.limit {
		return false
	}

	l.timestamps = append(l.timestamps, now)
	return true
}

// Count returns the number of requests currently in the window.
func (l *Log) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.timestamps)
}

// CountAt returns the number of requests in the window at the given time.
func (l *Log) CountAt(now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	return len(l.timestamps)
}

// Reset clears all tracked timestamps.
func (l *Log) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timestamps = l.timestamps[:0]
}

// Limit returns the configured request limit.
func (l *Log) Limit() int {
	return l.limit
}

// Window returns the configured window duration.
func (l *Log) Window() time.Duration {
	return l.window
}

// Oldest returns the oldest timestamp in the window.
// Returns zero time if no requests have been recorded.
func (l *Log) Oldest() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.timestamps) == 0 {
		return time.Time{}
	}
	return l.timestamps[0]
}
