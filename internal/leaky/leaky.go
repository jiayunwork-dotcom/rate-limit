// Package leaky implements a leaky bucket rate limiter.
//
// The leaky bucket algorithm processes requests at a fixed rate,
// allowing bursts up to the bucket capacity while draining at
// a constant rate.
package leaky

import (
	"sync"
	"time"
)

// Bucket represents a leaky bucket rate limiter.
// It allows requests at a fixed rate with burst capacity.
type Bucket struct {
	// Rate is the number of requests allowed per second.
	Rate float64
	// Capacity is the maximum number of requests that can be queued.
	Capacity float64

	mu        sync.Mutex
	water     float64   // 当前桶中的水量
	lastLeak  time.Time // 上次漏水时间
	clock     func() time.Time
}

// New creates a new leaky bucket with the given rate (requests/second)
// and capacity (maximum burst size).
func New(rate, capacity float64) *Bucket {
	return &Bucket{
		Rate:     rate,
		Capacity: capacity,
		water:    0,
		lastLeak: time.Now(),
		clock:    time.Now,
	}
}

// NewWithClock creates a new leaky bucket with a custom clock function
// for testing purposes.
func NewWithClock(rate, capacity float64, clock func() time.Time) *Bucket {
	return &Bucket{
		Rate:     rate,
		Capacity: capacity,
		water:    0,
		lastLeak: clock(),
		clock:    clock,
	}
}

// leak drains water from the bucket based on elapsed time.
func (b *Bucket) leak() {
	now := b.clock()
	elapsed := now.Sub(b.lastLeak).Seconds()
	if elapsed <= 0 {
		return
	}
	leaked := elapsed * b.Rate
	b.water -= leaked
	if b.water < 0 {
		b.water = 0
	}
	b.lastLeak = now
}

// Allow checks if a single request is allowed through the bucket.
// Returns true if the request is allowed, false if the bucket is full.
func (b *Bucket) Allow() bool {
	return b.AllowN(1)
}

// AllowN checks if n requests are allowed through the bucket.
// Returns true if there is enough capacity, false otherwise.
func (b *Bucket) AllowN(n int) bool {
	if n <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	b.leak()

	requested := float64(n)
	if b.water+requested > b.Capacity {
		return false
	}
	applyFill(b, requested)
	return true
}

// Wait returns the duration to wait before the next request can be processed.
// Returns zero if a request can be processed immediately.
func (b *Bucket) Wait() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.leak()

	if b.water < b.Capacity {
		return 0
	}

	// 计算需要等待多少时间才能腾出一个位置
	excess := b.water - b.Capacity + 1
	waitSeconds := excess / b.Rate
	return time.Duration(waitSeconds * float64(time.Second))
}

// WaterLevel returns the current water level in the bucket.
func (b *Bucket) WaterLevel() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.leak()
	return b.water
}
