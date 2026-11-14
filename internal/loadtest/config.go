package loadtest

import (
	"errors"
	"fmt"
	"time"
)

const (
	MaxConcurrency = 25
	MaxDuration    = 30 * time.Second
	MaxRPS         = 50
	MaxRequests    = 5_000
)

type Config struct {
	URL                 string        `json:"url"`
	Concurrency         int           `json:"concurrency"`
	Duration            time.Duration `json:"-"`
	RequestsPerSecond   int           `json:"requests_per_second"`
	MaxRequests         int           `json:"max_requests"`
	RequestTimeout      time.Duration `json:"-"`
	Authorization       bool          `json:"authorization_confirmed"`
	AllowPrivateTargets bool          `json:"-"`
}

func DefaultConfig() Config {
	return Config{Concurrency: 5, Duration: 10 * time.Second, RequestsPerSecond: 10, MaxRequests: 1000, RequestTimeout: 5 * time.Second}
}

func (c Config) Validate() error {
	if !c.Authorization {
		return errors.New("confirm that you own the target or have explicit permission to test it")
	}
	if c.Concurrency < 1 || c.Concurrency > MaxConcurrency {
		return fmt.Errorf("concurrency must be between 1 and %d", MaxConcurrency)
	}
	if c.Duration < time.Second || c.Duration > MaxDuration {
		return fmt.Errorf("duration must be between 1s and %s", MaxDuration)
	}
	if c.RequestsPerSecond < 1 || c.RequestsPerSecond > MaxRPS {
		return fmt.Errorf("requests_per_second must be between 1 and %d", MaxRPS)
	}
	if c.MaxRequests < 1 || c.MaxRequests > MaxRequests {
		return fmt.Errorf("max_requests must be between 1 and %d", MaxRequests)
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > 10*time.Second {
		return errors.New("request timeout must be greater than 0 and no more than 10s")
	}
	if c.URL == "" {
		return errors.New("url is required")
	}
	return nil
}
