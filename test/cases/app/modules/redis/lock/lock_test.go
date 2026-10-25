package lock

import (
	"context"
	"testing"
	"time"
	"willshark/app/modules/redis"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newStore(t *testing.T) (*redis.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return &redis.Redis{Cache: client}, mr
}

func TestRedisLockMutualExclusion(t *testing.T) {
	store, mr := newStore(t)
	ctx := context.Background()

	first := NewRedisLock(store, "k")
	first.SetExpire(5)
	ok, err := first.AcquireCtx(ctx)
	if err != nil || !ok {
		t.Fatalf("first acquire: ok=%v err=%v", ok, err)
	}

	// 锁被占用时应返回 (false, nil)，而不是报错
	second := NewRedisLock(store, "k")
	second.SetExpire(5)
	ok, err = second.AcquireCtx(ctx)
	if err != nil || ok {
		t.Fatalf("second acquire should fail without error: ok=%v err=%v", ok, err)
	}

	// 同一持有者可重入并续期
	ok, err = first.AcquireCtx(ctx)
	if err != nil || !ok {
		t.Fatalf("reentrant acquire: ok=%v err=%v", ok, err)
	}

	// 非持有者不能释放
	released, err := second.ReleaseCtx(ctx)
	if err != nil || released {
		t.Fatalf("non-owner release: released=%v err=%v", released, err)
	}

	released, err = first.ReleaseCtx(ctx)
	if err != nil || !released {
		t.Fatalf("owner release: released=%v err=%v", released, err)
	}

	ok, err = second.AcquireCtx(ctx)
	if err != nil || !ok {
		t.Fatalf("acquire after release: ok=%v err=%v", ok, err)
	}

	// 过期后其他持有者可获取
	mr.FastForward(6 * time.Second)
	ok, err = first.AcquireCtx(ctx)
	if err != nil || !ok {
		t.Fatalf("acquire after expire: ok=%v err=%v", ok, err)
	}
}

func TestRedisLockHonorsContext(t *testing.T) {
	store, _ := newStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	l := NewRedisLock(store, "k")
	l.SetExpire(5)
	if _, err := l.AcquireCtx(ctx); err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestLockWithoutContextAndErrors(t *testing.T) {
	store, mr := newStore(t)
	l := NewRedisLock(store, "plain")
	l.SetExpire(-5) // 负数按 0 处理，不能回绕成超长 TTL
	if ok, err := l.Acquire(); !ok || err != nil {
		t.Fatalf("acquire: %v %v", ok, err)
	}
	if ttl := mr.TTL("plain"); ttl <= 0 || ttl > time.Second {
		t.Fatalf("ttl = %v, want ~500ms", ttl)
	}
	if ok, err := l.Release(); !ok || err != nil {
		t.Fatalf("release: %v %v", ok, err)
	}
	if ok, _ := l.Release(); ok {
		t.Fatal("second release must report false")
	}
	if a, b := NewRedisLock(store, "x").id, NewRedisLock(store, "x").id; a == b || len(a) != randomLen {
		t.Fatalf("lock ids must be random hex of len %d: %q %q", randomLen, a, b)
	}
	mr.Close()
	if _, err := l.ReleaseCtx(context.Background()); err == nil {
		t.Fatal("release with redis down must return error")
	}
	if _, err := l.AcquireCtx(context.Background()); err == nil {
		t.Fatal("acquire with redis down must return error")
	}
}
