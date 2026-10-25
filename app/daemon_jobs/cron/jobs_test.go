package cron

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestEveryRunsAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var j Jobs
		var runs atomic.Int32
		ctx, cancel := context.WithCancel(context.Background())
		j.every(ctx, "count", time.Minute, func(context.Context) { runs.Add(1) })

		time.Sleep(3*time.Minute + time.Second)
		synctest.Wait()
		if got := runs.Load(); got != 3 {
			t.Fatalf("runs = %d, want 3", got)
		}

		cancel()
		if err := j.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Minute)
		if got := runs.Load(); got != 3 {
			t.Fatalf("job ran after stop: %d", got)
		}
	})
}

func TestPanicDoesNotKillLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var j Jobs
		var runs atomic.Int32
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		j.every(ctx, "panicky", time.Minute, func(context.Context) {
			runs.Add(1)
			panic("boom")
		})
		time.Sleep(2*time.Minute + time.Second)
		synctest.Wait()
		if got := runs.Load(); got != 2 {
			t.Fatalf("runs = %d, want 2 (loop must survive panics)", got)
		}
		cancel()
		_ = j.Wait(context.Background())
	})
}

func TestWaitTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var j Jobs
		release := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		j.every(ctx, "slow", time.Second, func(context.Context) { <-release })
		time.Sleep(time.Second + time.Millisecond) // 任务进入执行
		cancel()

		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer waitCancel()
		if err := j.Wait(waitCtx); err == nil {
			t.Fatal("Wait should time out while a job is still running")
		}
		close(release)
		_ = j.Wait(context.Background())
	})
}
