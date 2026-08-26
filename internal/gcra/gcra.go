package gcra

import (
	"sync"
	"time"
)

type GCRA struct {
	mu       sync.Mutex
	rate     float64
	burst    int
	emission time.Duration
	delay    time.Duration
	tat      time.Time
}

func NewGCRA(rate float64, burst int) *GCRA {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = 1
	}

	emission := time.Duration(float64(time.Second) / rate)
	delay := emission * time.Duration(burst)

	return &GCRA{
		rate:     rate,
		burst:    burst,
		emission: emission,
		delay:    delay,
	}
}

func (g *GCRA) Allow(now time.Time) bool {
	return g.AllowN(now, 1)
}

func (g *GCRA) AllowN(now time.Time, n int) bool {
	if n <= 0 {
		return true
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	increment := g.emission * time.Duration(n)

	if g.tat.IsZero() {
		newTAT := now.Add(increment)
		limit := now.Add(g.delay)
		if newTAT.After(limit) {
			return false
		}
		g.tat = newTAT
		return true
	}

	newTAT := g.tat.Add(increment)

	if now.After(g.tat) {
		newTAT = now.Add(increment)
	}

	limit := now.Add(g.delay)
	if newTAT.After(limit) {
		return false
	}

	g.tat = newTAT
	return true
}

func (g *GCRA) Rate() float64 {
	return g.rate
}

func (g *GCRA) Burst() int {
	return g.burst
}

func (g *GCRA) Emission() time.Duration {
	return g.emission
}

func (g *GCRA) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tat = time.Time{}
}

func (g *GCRA) TAT() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.tat
}
