package lock

import (
	"context"
	"testing"
	"time"
)

func TestLockerExclusiveAndRelease(t *testing.T) {
	store, mr := newStore(t)
	l := NewLocker(store)
	ctx := context.Background()

	unlock, ok, err := l.TryLock(ctx, "cron:job", 1500*time.Millisecond)
	if err != nil || !ok {
		t.Fatalf("first: ok=%v err=%v", ok, err)
	}
	if ttl := mr.TTL("cron:job"); ttl < 2*time.Second {
		t.Fatalf("ttl should be rounded up to whole seconds (+tolerance), got %v", ttl)
	}
	if _, ok, err := l.TryLock(ctx, "cron:job", time.Second); err != nil || ok {
		t.Fatalf("second instance must not get the lock: ok=%v err=%v", ok, err)
	}
	unlock(ctx)
	if mr.Exists("cron:job") {
		t.Fatal("unlock should delete the key")
	}
	if _, ok, _ := l.TryLock(ctx, "cron:job", time.Second); !ok {
		t.Fatal("lock should be available after unlock")
	}
}
