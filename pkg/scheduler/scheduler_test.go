package scheduler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func counter() (*atomic.Int32, Func) {
	var n atomic.Int32
	return &n, func(context.Context) error { n.Add(1); return nil }
}

func TestEveryRunsUntilCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		n, fn := counter()
		s.Every("count", time.Minute, fn)
		ctx, cancel := context.WithCancel(context.Background())
		s.Start(ctx)

		time.Sleep(3*time.Minute + time.Second)
		synctest.Wait()
		if got := n.Load(); got != 3 {
			t.Fatalf("runs = %d, want 3", got)
		}
		cancel()
		if err := s.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Minute)
		if got := n.Load(); got != 3 {
			t.Fatalf("ran after cancel: %d", got)
		}
	})
}

func TestCronSpec(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		var mu sync.Mutex
		var at []time.Time
		s.Cron("every15s", "*/15 * * * * *", func(context.Context) error {
			mu.Lock()
			at = append(at, time.Now())
			mu.Unlock()
			return nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		time.Sleep(time.Minute)
		synctest.Wait()
		mu.Lock()
		defer mu.Unlock()
		if len(at) != 4 {
			t.Fatalf("runs = %d, want 4", len(at))
		}
		for _, ts := range at {
			if ts.Second()%15 != 0 {
				t.Fatalf("run at %v not aligned to 15s", ts)
			}
		}
	})
}

func TestRunOnStartAndTrigger(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		n, fn := counter()
		s.Every("job", time.Hour, fn, WithRunOnStart())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		synctest.Wait()
		if n.Load() != 1 {
			t.Fatalf("run on start: %d", n.Load())
		}
		if err := s.Trigger("job"); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if n.Load() != 2 {
			t.Fatalf("after trigger: %d", n.Load())
		}
		if err := s.Trigger("nope"); !errors.Is(err, ErrUnknownJob) {
			t.Fatalf("unknown job err = %v", err)
		}
	})
}

func TestTriggersCoalesceWhileRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		var n atomic.Int32
		release := make(chan struct{})
		s.Every("slow", time.Hour, func(context.Context) error {
			n.Add(1)
			<-release
			return nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		_ = s.Trigger("slow")
		synctest.Wait() // 第一次执行中
		for range 5 {
			_ = s.Trigger("slow")
		}
		close(release)
		synctest.Wait()
		if got := n.Load(); got != 2 {
			t.Fatalf("runs = %d, want 2 (triggers during a run coalesce)", got)
		}
	})
}

type fakeLocker struct {
	mu       sync.Mutex
	held     map[string]bool
	err      error
	unlocked []string
}

func (l *fakeLocker) TryLock(_ context.Context, key string, _ time.Duration) (func(context.Context), bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return nil, false, l.err
	}
	if l.held[key] {
		return nil, false, nil
	}
	l.held[key] = true
	return func(context.Context) {
		l.mu.Lock()
		defer l.mu.Unlock()
		delete(l.held, key)
		l.unlocked = append(l.unlocked, key)
	}, true, nil
}

func TestLockSkipsWhenHeldElsewhere(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := &fakeLocker{held: map[string]bool{"app:locked": true}}
		s := New(WithLocker(l), WithLockPrefix("app:"))
		locked, fn1 := counter()
		free, fn2 := counter()
		s.Every("locked", time.Minute, fn1, WithLock(time.Minute))
		s.Every("free", time.Minute, fn2, WithLock(time.Minute))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		time.Sleep(2*time.Minute + time.Second)
		synctest.Wait()
		if locked.Load() != 0 || free.Load() != 2 {
			t.Fatalf("locked=%d free=%d", locked.Load(), free.Load())
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		if len(l.unlocked) != 2 || l.unlocked[0] != "app:free" {
			t.Fatalf("lock must be released after each run: %v", l.unlocked)
		}
	})
}

func TestLockErrorSkipsRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New(WithLocker(&fakeLocker{held: map[string]bool{}, err: errors.New("redis down")}))
		n, fn := counter()
		s.Every("j", time.Minute, fn, WithLock(time.Minute))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		s.Start(ctx)
		time.Sleep(time.Minute + time.Second)
		synctest.Wait()
		if n.Load() != 0 {
			t.Fatal("job must not run when the lock cannot be checked")
		}
	})
}

func TestTimeoutAndPanicDoNotKillLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		var timedOut, panics atomic.Int32
		s.Every("slow", time.Minute, func(ctx context.Context) error {
			<-ctx.Done()
			timedOut.Add(1)
			return ctx.Err()
		}, WithTimeout(5*time.Second))
		s.Every("panicky", time.Minute, func(context.Context) error {
			panics.Add(1)
			panic("boom")
		})
		ctx, cancel := context.WithCancel(context.Background())
		s.Start(ctx)
		time.Sleep(3 * time.Minute)
		synctest.Wait()
		if timedOut.Load() < 2 || panics.Load() != 3 {
			t.Fatalf("timedOut=%d panics=%d", timedOut.Load(), panics.Load())
		}
		cancel()
		_ = s.Wait(context.Background())
	})
}

func TestWaitTimesOutOnStuckJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		release := make(chan struct{})
		s.Every("stuck", time.Second, func(context.Context) error { <-release; return nil }, WithRunOnStart())
		ctx, cancel := context.WithCancel(context.Background())
		s.Start(ctx)
		synctest.Wait()
		cancel()
		wctx, wcancel := context.WithTimeout(context.Background(), time.Second)
		defer wcancel()
		if err := s.Wait(wctx); err == nil {
			t.Fatal("Wait should time out")
		}
		close(release)
		_ = s.Wait(context.Background())
	})
}

func TestRegistrationErrorsPanic(t *testing.T) {
	cases := map[string]func(s *Scheduler){
		"duplicate":      func(s *Scheduler) { s.Every("a", time.Second, nil); s.Every("a", time.Second, nil) },
		"bad interval":   func(s *Scheduler) { s.Every("a", 0, nil) },
		"bad cron":       func(s *Scheduler) { s.Cron("a", "not a spec", nil) },
		"lock no locker": func(s *Scheduler) { s.Every("a", time.Second, nil, WithLock(time.Second)) },
		"after start": func(s *Scheduler) {
			ctx, cancel := context.WithCancel(context.Background())
			s.Start(ctx)
			cancel()
			s.Every("late", time.Second, nil)
		},
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(r.(string), "scheduler:") {
					t.Fatalf("expected scheduler panic, got %v", r)
				}
			}()
			fn(New())
		})
	}
}

func TestNames(t *testing.T) {
	s := New()
	s.Every("b", time.Second, nil)
	s.Cron("a", "@daily", nil)
	if got := strings.Join(s.Names(), ","); got != "a,b" {
		t.Fatalf("names = %s", got)
	}
}
