package retry

import (
	"errors"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	max := 10 * time.Second

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		result := ExponentialBackoff(tt.attempt, base, max)
		if result != tt.expected {
			t.Errorf("ExponentialBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
		}
	}

	result := ExponentialBackoff(100, base, max)
	if result > max {
		t.Errorf("ExponentialBackoff(100) = %v, should not exceed max %v", result, max)
	}
}

func TestLinearBackoff(t *testing.T) {
	base := 100 * time.Millisecond

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 300 * time.Millisecond},
		{3, 400 * time.Millisecond},
	}

	for _, tt := range tests {
		result := LinearBackoff(tt.attempt, base)
		if result != tt.expected {
			t.Errorf("LinearBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
		}
	}
}

func TestWithJitter(t *testing.T) {
	d := time.Second

	if result := WithJitter(d, 0); result != d {
		t.Errorf("WithJitter(0) = %v, want %v", result, d)
	}

	if result := WithJitter(d, 1.5); result != d {
		t.Errorf("WithJitter(1.5) = %v, want %v", result, d)
	}

	for i := 0; i < 100; i++ {
		result := WithJitter(d, 0.5)
		min := time.Duration(float64(d) * 0.5)
		max := time.Duration(float64(d) * 1.5)
		if result < min || result > max {
			t.Errorf("WithJitter(0.5) = %v, not in [%v, %v]", result, min, max)
		}
	}
}

func TestRetrySuccess(t *testing.T) {
	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("not ready")
		}
		return nil
	}

	backoff := Backoff{
		Base: time.Millisecond,
		Max:  10 * time.Millisecond,
	}

	err := Retry(5, backoff, fn)
	if err != nil {
		t.Errorf("Retry() error = %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryMaxAttempts(t *testing.T) {
	fn := func() error {
		return errors.New("always fails")
	}

	backoff := Backoff{
		Base: time.Millisecond,
		Max:  10 * time.Millisecond,
	}

	err := Retry(3, backoff, fn)
	if err == nil {
		t.Error("Retry() should return error when all attempts fail")
	}
}

func TestRetryZeroAttempts(t *testing.T) {
	backoff := DefaultBackoff()
	err := Retry(0, backoff, func() error { return nil })
	if err != ErrMaxAttempts {
		t.Errorf("Retry(0) error = %v, want ErrMaxAttempts", err)
	}
}
