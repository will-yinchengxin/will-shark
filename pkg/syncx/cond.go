package syncx

import "time"

type Cond struct {
	signal chan struct{}
}

func NewCond() *Cond {
	return &Cond{
		signal: make(chan struct{}),
	}
}

// Signal 唤醒一个等待者；没有等待者时不阻塞
func (cond *Cond) Signal() {
	select {
	case cond.signal <- struct{}{}:
	default:
	}
}

// WaitWithTimeout 等待 Signal，返回剩余超时时间与是否被唤醒
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
