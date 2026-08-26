package retry

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

var ErrMaxAttempts = errors.New("retry: maximum attempts reached")

type Backoff struct {
	Base   time.Duration
	Max    time.Duration
	Factor float64
	Jitter float64
}

func DefaultBackoff() Backoff {
	return Backoff{
		Base:   100 * time.Millisecond,
		Max:    30 * time.Second,
		Factor: 2.0,
		Jitter: 0.1,
	}
}

func ExponentialBackoff(attempt int, base, max time.Duration) time.Duration {
	if attempt <= 0 {
		return base
	}

	multiplier := math.Pow(2, float64(attempt))
	duration := time.Duration(float64(base) * multiplier)

	if duration > max || duration <= 0 {
		return max
	}
	return duration
}

func LinearBackoff(attempt int, base time.Duration) time.Duration {
	if attempt <= 0 {
		return base
	}
	return base * time.Duration(attempt+1)
}

func WithJitter(d time.Duration, factor float64) time.Duration {
	if factor <= 0 || factor > 1 {
		return d
	}

	jitter := (rand.Float64()*2 - 1) * factor
	return time.Duration(float64(d) * (1 + jitter))
}

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

		if attempt < maxAttempts-1 {
			wait := calculateWait(attempt, backoff)
			time.Sleep(wait)
		}
	}

	return lastErr
}

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
