package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const (
	AlgorithmTokenBucket = "token_bucket"
	AlgorithmLeakyBucket = "leaky_bucket"
	AlgorithmFixedWindow = "fixed_window"
	AlgorithmSlidingLog  = "sliding_log"
	AlgorithmGCRA        = "gcra"
)

type Config struct {
	Algorithm string  `json:"algorithm"`
	Rate      float64 `json:"rate"`
	Burst     int     `json:"burst"`
	Window    string  `json:"window"`
	Limit     int     `json:"limit"`
	Name      string  `json:"name,omitempty"`
	Enabled   bool    `json:"enabled"`
}

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

func Parse(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("config: failed to parse: %w", err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.Algorithm == "" {
		errs = append(errs, errors.New("algorithm is required"))
	} else {
		switch c.Algorithm {
		case AlgorithmTokenBucket, AlgorithmLeakyBucket,
			AlgorithmFixedWindow, AlgorithmSlidingLog, AlgorithmGCRA:
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

	msg := "config validation failed:"
	for _, e := range errs {
		msg += " " + e.Error() + ";"
	}
	return errors.New(msg)
}

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

func (c Config) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
