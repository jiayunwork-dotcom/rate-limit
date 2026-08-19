// Package fixed implements a fixed window counter rate limiter.
//
// The fixed window algorithm divides time into fixed-size windows
// and counts requests within each window. When the count exceeds
// the limit, subsequent requests are denied until the next window.
package fixed

import (
	"sync"
	"time"
)

// Counter implements a fixed window counter rate limiter.
type Counter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	count       int
	windowStart time.Time
}

// NewCounter creates a new fixed window counter.
// limit specifies the maximum requests per window.
// window specifies the duration of each fixed window.
func NewCounter(limit int, window time.Duration) *Counter {
	return &Counter{
		limit:  limit,
		window: window,
	}
}

// advance checks if the current window has expired and resets if needed.
func (c *Counter) advance(now time.Time) {
	if c.windowStart.IsZero() {
		c.windowStart = now
		c.count = 0
		return
	}
	elapsed := now.Sub(c.windowStart)
	if elapsed >= c.window {
		c.windowStart = now
		c.count = 0
	}
}

// Allow checks if a request at the given time is allowed.
// Returns true if the request count within the current window is below the limit.
func (c *Counter) Allow(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.advance(now)

	if c.count >= c.limit {
		return false
	}
	c.count++
	return true
}

// Remaining returns the number of remaining allowed requests in the current window.
func (c *Counter) Remaining() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	remaining := c.limit - c.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset clears the counter and window state.
func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count = 0
	c.windowStart = time.Time{}
}

// WindowStart returns the start time of the current window.
func (c *Counter) WindowStart() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.windowStart
}

// Limit returns the configured request limit per window.
func (c *Counter) Limit() int {
	return c.limit
}
