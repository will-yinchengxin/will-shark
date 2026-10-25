package syncx

import (
	"testing"
	"time"
)

func TestPoolWaitDoesNotHangAfterFailedSubmit(t *testing.T) {
	p, err := NewPool(1)
	if err != nil {
		t.Fatal(err)
	}
	p.Release()
	if err := p.Submit(func() {}); err == nil {
		t.Fatal("submit to a released pool should fail")
	}
	done := make(chan struct{})
	go func() { p.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait hangs after a failed Submit")
	}
}

func TestCondSignal(t *testing.T) {
	c := NewCond()
	go func() {
		time.Sleep(10 * time.Millisecond)
		c.Signal()
	}()
	if _, ok := c.WaitWithTimeout(time.Second); !ok {
		t.Fatal("Signal must wake the waiter")
	}
	c.Signal() // 无等待者时不阻塞
}
