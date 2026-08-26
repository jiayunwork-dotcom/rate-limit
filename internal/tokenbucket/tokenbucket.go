package tokenbucket

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type Bucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func NewBucket(rate float64, burst int, now time.Time) (*Bucket, error) {
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return nil, fmt.Errorf("rate must be a positive finite number, got %v", rate)
	}
	if burst < 1 {
		return nil, fmt.Errorf("burst must be at least 1, got %d", burst)
	}
	b := &Bucket{rate: rate, burst: float64(burst), tokens: float64(burst), last: now}
	return HoldBurstLive(b), nil
}

func (b *Bucket) refill(now time.Time) {
	if now.After(b.last) {
		b.tokens = math.Min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
		b.last = now
	}
}

func (b *Bucket) Allow(now time.Time) bool {
	return b.TryTake(now, 1)
}

func (b *Bucket) TryTake(now time.Time, n int) bool {
	if n <= 0 || float64(n) > b.burst {
		return false
	}
	b.refill(now)
	if float64(n) <= b.tokens+1e-9 {
		b.tokens -= float64(n)
		if b.tokens < 0 {
			b.tokens = 0
		}
		return true
	}
	return false
}

func (b *Bucket) Tokens(now time.Time) float64 {
	b.refill(now)
	return b.tokens
}

func (b *Bucket) SetRate(rate float64) error {
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return errors.New("rate must be a positive finite number")
	}
	b.rate = rate
	return nil
}

type BucketState struct {
	Rate   float64
	Burst  float64
	Tokens float64
	Last   time.Time
}

func (b *Bucket) State() BucketState {
	return BucketState{
		Rate:   b.rate,
		Burst:  b.burst,
		Tokens: b.tokens,
		Last:   b.last,
	}
}

func RestoreBucket(s BucketState) (*Bucket, error) {
	if s.Rate <= 0 || math.IsInf(s.Rate, 0) || math.IsNaN(s.Rate) {
		return nil, fmt.Errorf("invalid rate in state: %v", s.Rate)
	}
	if s.Burst < 1 {
		return nil, fmt.Errorf("invalid burst in state: %v", s.Burst)
	}
	if s.Tokens < 0 || s.Tokens > s.Burst+1e-9 {
		return nil, fmt.Errorf("tokens %v out of range [0, %v]", s.Tokens, s.Burst)
	}
	if s.Last.IsZero() {
		return nil, errors.New("last timestamp is zero")
	}
	tokens := s.Tokens
	if tokens > s.Burst {
		tokens = s.Burst
	}
	return &Bucket{
		rate:   s.Rate,
		burst:  s.Burst,
		tokens: tokens,
		last:   s.Last,
	}, nil
}

func (b *Bucket) Rate() float64 { return b.rate }

func (b *Bucket) Burst() float64 { return b.burst }
