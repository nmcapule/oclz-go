package scheduler

import "time"

type LoopConfig struct {
	InitialWait time.Duration
	RetryWait   time.Duration
}

func Launch(fn func(quit chan struct{}), config LoopConfig) error {
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
