// Package tokenbucket implements a token bucket rate limiter with continuous
// refill.
package tokenbucket

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Bucket holds at most Burst tokens and refills at Rate tokens per second.
// All timestamps come from the caller, so behavior is deterministic.
type Bucket struct {
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

// NewBucket creates a full bucket at time now. Rate is tokens per second and
// must be positive; burst must be at least one.
func NewBucket(rate float64, burst int, now time.Time) (*Bucket, error) {
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return nil, fmt.Errorf("rate must be a positive finite number, got %v", rate)
	}
	if burst < 1 {
		return nil, fmt.Errorf("burst must be at least 1, got %d", burst)
	}
	return &Bucket{rate: rate, burst: float64(burst), tokens: float64(burst), last: now}, nil
}

func (b *Bucket) refill(now time.Time) {
	if now.After(b.last) {
		b.tokens = math.Min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
		b.last = now
	}
}

// Allow takes one token if available.
func (b *Bucket) Allow(now time.Time) bool {
	return b.TryTake(now, 1)
}

// TryTake takes n tokens if available. n must be positive.
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

// Tokens reports the current token count after refilling to now.
func (b *Bucket) Tokens(now time.Time) float64 {
	b.refill(now)
	return b.tokens
}

// SetRate changes the refill rate (tokens per second, positive).
func (b *Bucket) SetRate(rate float64) error {
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return errors.New("rate must be a positive finite number")
	}
	b.rate = rate
	return nil
}

// BucketState holds the serializable state of a token bucket.
type BucketState struct {
	Rate   float64
	Burst  float64
	Tokens float64
	Last   time.Time
}

// State returns the current bucket state for serialization.
func (b *Bucket) State() BucketState {
	return BucketState{
		Rate:   b.rate,
		Burst:  b.burst,
		Tokens: b.tokens,
		Last:   b.last,
	}
}

// RestoreBucket creates a Bucket from a previously saved state. It validates
// the state and returns an error if the values are inconsistent.
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

// Rate returns the current refill rate (tokens per second).
func (b *Bucket) Rate() float64 { return b.rate }

// Burst returns the maximum token capacity.
func (b *Bucket) Burst() float64 { return b.burst }
