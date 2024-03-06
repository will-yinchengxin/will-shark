package utils

import "time"

type Cond struct {
	signal chan struct{}
}

func NewCond() *Cond {
	return &Cond{
		signal: make(chan struct{}),
	}
}

func (cond *Cond) WaitWithTimeout(timeout time.Duration) (time.Duration, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	begin := time.Now()
	select {
	case <-cond.signal:
		elapsed := time.Since(begin)
		remainTimeout := timeout - elapsed
		return remainTimeout, true
	case <-timer.C:
		return 0, false
	}
}
