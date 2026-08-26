package leaky

import (
	"sync"
	"time"
)

type Bucket struct {
	Rate     float64
	Capacity float64

	mu       sync.Mutex
	water    float64
	lastLeak time.Time
	clock    func() time.Time
}

func New(rate, capacity float64) *Bucket {
	return &Bucket{
		Rate:     rate,
		Capacity: capacity,
		water:    0,
		lastLeak: time.Now(),
		clock:    time.Now,
	}
}

func NewWithClock(rate, capacity float64, clock func() time.Time) *Bucket {
	return &Bucket{
		Rate:     rate,
		Capacity: capacity,
		water:    0,
		lastLeak: clock(),
		clock:    clock,
	}
}

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

func (b *Bucket) Allow() bool {
	return b.AllowN(1)
}

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
	b.water += requested
	return true
}

func (b *Bucket) Wait() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.leak()

	if b.water < b.Capacity {
		return 0
	}

	excess := b.water - b.Capacity + 1
	waitSeconds := excess / b.Rate
	return time.Duration(waitSeconds * float64(time.Second))
}

func (b *Bucket) WaterLevel() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.leak()
	return b.water
}
