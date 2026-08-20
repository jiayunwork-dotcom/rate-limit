// Package config provides configuration parsing and validation
// for rate limiters.
//
// It supports JSON-based configuration with sensible defaults
// and comprehensive validation.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// 支持的算法类型
const (
	AlgorithmTokenBucket  = "token_bucket"
	AlgorithmLeakyBucket  = "leaky_bucket"
	AlgorithmFixedWindow  = "fixed_window"
	AlgorithmSlidingLog   = "sliding_log"
	AlgorithmGCRA         = "gcra"
)

// Config holds the configuration for a rate limiter.
type Config struct {
	// Algorithm specifies which rate limiting algorithm to use.
	Algorithm string `json:"algorithm"`
	// Rate is the number of requests allowed per second.
	Rate float64 `json:"rate"`
	// Burst is the maximum burst size.
	Burst int `json:"burst"`
	// Window is the duration string for window-based algorithms (e.g., "1m", "1h").
	Window string `json:"window"`
	// Limit is the maximum requests per window.
	Limit int `json:"limit"`
	// Name is an optional identifier for this configuration.
	Name string `json:"name,omitempty"`
	// Enabled indicates whether the rate limiter is active.
	Enabled bool `json:"enabled"`
}

// Default returns a default configuration using the token bucket algorithm.
func Default() Config {
	return Config{
		Algorithm: AlgorithmTokenBucket,
		Rate:      100,
		Burst:     10,
		Window:    "1m",
		Limit:     1000,
		Enabled:   true,
	}
}

// Parse parses a JSON configuration from the given data.
func Parse(data []byte) (Config, error) {
	return commitParse(data)
}

// Validate checks if the configuration is valid.
// Returns an error describing any validation failures.
func (c Config) Validate() error {
	var errs []error

	if c.Algorithm == "" {
		errs = append(errs, errors.New("algorithm is required"))
	} else {
		switch c.Algorithm {
		case AlgorithmTokenBucket, AlgorithmLeakyBucket,
			AlgorithmFixedWindow, AlgorithmSlidingLog, AlgorithmGCRA:
			// 有效算法
		default:
			errs = append(errs, fmt.Errorf("unknown algorithm: %q", c.Algorithm))
		}
	}

	if c.Rate <= 0 {
		errs = append(errs, errors.New("rate must be positive"))
	}

	if c.Burst < 0 {
		errs = append(errs, errors.New("burst must be non-negative"))
	}

	if c.Window != "" {
		if _, err := time.ParseDuration(c.Window); err != nil {
			errs = append(errs, fmt.Errorf("invalid window duration: %w", err))
		}
	}

	if c.Limit < 0 {
		errs = append(errs, errors.New("limit must be non-negative"))
	}

	if len(errs) == 0 {
		return nil
	}

	// 合并所有错误
	msg := "config validation failed:"
	for _, e := range errs {
		msg += " " + e.Error() + ";"
	}
	return errors.New(msg)
}

// WindowDuration parses and returns the Window field as a time.Duration.
// Returns zero if the window is empty or invalid.
func (c Config) WindowDuration() time.Duration {
	if c.Window == "" {
		return 0
	}
	d, err := time.ParseDuration(c.Window)
	if err != nil {
		return 0
	}
	return d
}

// ToJSON serializes the configuration to JSON.
func (c Config) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
