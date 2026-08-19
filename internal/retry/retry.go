// Package retry implements retry logic with various backoff strategies.
//
// It supports exponential backoff, linear backoff, and jitter
// to help avoid thundering herd problems.
package retry

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// ErrMaxAttempts is returned when the maximum number of retry attempts is reached.
var ErrMaxAttempts = errors.New("retry: maximum attempts reached")

// Backoff defines the configuration for retry backoff.
type Backoff struct {
	// Base is the initial backoff duration.
	Base time.Duration
	// Max is the maximum backoff duration.
	Max time.Duration
	// Factor is the multiplication factor for exponential backoff.
	Factor float64
	// Jitter is the jitter factor (0.0 to 1.0) added to backoff durations.
	Jitter float64
}

// DefaultBackoff returns a sensible default backoff configuration.
func DefaultBackoff() Backoff {
	return Backoff{
		Base:   100 * time.Millisecond,
		Max:    30 * time.Second,
		Factor: 2.0,
		Jitter: 0.1,
	}
}

// ExponentialBackoff calculates the backoff duration for a given attempt
// using exponential growth with the given base and max durations.
func ExponentialBackoff(attempt int, base, max time.Duration) time.Duration {
	if attempt <= 0 {
		return base
	}

	// 计算指数退避
	multiplier := math.Pow(2, float64(attempt))
	duration := time.Duration(float64(base) * multiplier)

	if duration > max || duration <= 0 {
		return max
	}
	return duration
}

// LinearBackoff calculates the backoff duration for a given attempt
// using linear growth with the given base duration.
func LinearBackoff(attempt int, base time.Duration) time.Duration {
	if attempt <= 0 {
		return base
	}
	return base * time.Duration(attempt+1)
}

// WithJitter adds random jitter to a duration.
// factor controls the range: result will be in [d*(1-factor), d*(1+factor)].
func WithJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 || factor > 1 {
		return d
	}

	// 生成 [-factor, +factor] 范围的随机偏移
	jitter := (rand.Float64()*2 - 1) * factor
	return time.Duration(float64(d) * (1 + jitter))
}

// Retry executes the given function up to maxAttempts times with backoff
// between attempts. Returns nil if fn succeeds, or ErrMaxAttempts if
// all attempts fail.
func Retry(maxAttempts int, backoff Backoff, fn func() error) error {
	if maxAttempts <= 0 {
		return ErrMaxAttempts
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// 最后一次尝试不需要等待
		if attempt < maxAttempts-1 {
			wait := calculateWait(attempt, backoff)
			time.Sleep(wait)
		}
	}

	return lastErr
}

// RetryWithResult executes the given function up to maxAttempts times,
// returning both the result and any error.
func RetryWithResult[T any](maxAttempts int, backoff Backoff, fn func() (T, error)) (T, error) {
	var zero T
	if maxAttempts <= 0 {
		return zero, ErrMaxAttempts
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < maxAttempts-1 {
			wait := calculateWait(attempt, backoff)
			time.Sleep(wait)
		}
	}

	return zero, lastErr
}

// calculateWait computes the wait duration for a given attempt.
func calculateWait(attempt int, b Backoff) time.Duration {
	var wait time.Duration

	if b.Factor > 0 {
		multiplier := math.Pow(b.Factor, float64(attempt))
		wait = time.Duration(float64(b.Base) * multiplier)
	} else {
		wait = b.Base * time.Duration(attempt+1)
	}

	if wait > b.Max {
		wait = b.Max
	}

	if b.Jitter > 0 {
		wait = WithJitter(wait, b.Jitter)
	}

	return wait
}
