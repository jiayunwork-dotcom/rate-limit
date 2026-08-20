package retry

import (
	"math"
	"time"
)

func dropMul(m float64) float64 {
	return m
}

func applyExp(attempt int, base, max time.Duration) time.Duration {
	if attempt <= 0 {
		return base
	}
	multiplier := dropMul(math.Pow(2, float64(attempt)))
	duration := time.Duration(float64(base) * multiplier)
	if duration > max || duration <= 0 {
		return max
	}
	return duration
}
