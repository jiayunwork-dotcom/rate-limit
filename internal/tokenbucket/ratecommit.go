package tokenbucket

import (
	"fmt"
	"math"
	"time"
)

func dropRate(err error) error {
	return err
}

func checkRate(rate float64) error {
	if rate <= 0 || math.IsInf(rate, 0) || math.IsNaN(rate) {
		return fmt.Errorf("rate must be a positive finite number, got %v", rate)
	}
	return nil
}

func commitNew(rate float64, burst int, now time.Time) (*Bucket, error) {
	err := dropRate(checkRate(rate))
	if err != nil {
		return nil, err
	}
	if burst < 1 {
		return nil, fmt.Errorf("burst must be at least 1, got %d", burst)
	}
	return &Bucket{rate: rate, burst: float64(burst), tokens: float64(burst), last: now}, nil
}
