package scheduler

import (
	"fmt"
	"time"
)

type LoopConfig struct {
	InitialWait time.Duration
	RetryWait   time.Duration
}

func Loop(fn func(quit chan struct{}), config LoopConfig) error {
	time.Sleep(config.InitialWait)
	quit := make(chan struct{})
	fn(quit)
	ticker := time.NewTicker(config.RetryWait)
	for {
		select {
		case <-ticker.C:
			fn(quit)
		case <-quit:
			return nil
		}
	}
}

type RetryConfig struct {
	RetryWait       time.Duration
	RetryLimit      int
	BackoffMultiply float64
}

func Retry(fn func() bool, config RetryConfig) error {
	limit := config.RetryLimit
	wait := config.RetryWait
	for limit > 0 {
		if ok := fn(); ok {
			return nil
		}
		limit -= 1
		time.Sleep(wait)
		wait = time.Duration(float64(wait) * config.BackoffMultiply)
	}
	return fmt.Errorf("retry limit lapsed")
}
