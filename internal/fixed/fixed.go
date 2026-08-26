package fixed

import (
	"sync"
	"time"
)

type Counter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	count       int
	windowStart time.Time
}

func NewCounter(limit int, window time.Duration) *Counter {
	return &Counter{
		limit:  limit,
		window: window,
	}
}

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

func (c *Counter) Remaining() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	remaining := c.limit - c.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (c *Counter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count = 0
	c.windowStart = time.Time{}
}

func (c *Counter) WindowStart() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.windowStart
}

func (c *Counter) Limit() int {
	return c.limit
}
