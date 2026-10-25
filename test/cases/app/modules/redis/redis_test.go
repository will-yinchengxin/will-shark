package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// 与 di.InitRedis 的构造方式一致：只设置 Cache
func TestRedisWrapperBuiltLikeDI(t *testing.T) {
	mr := miniredis.RunT(t)
	r := &Redis{Cache: goredis.NewClient(&goredis.Options{Addr: mr.Addr()})}
	ctx := context.Background()

	if v, err := r.Get(ctx, "missing"); err != nil || v != "" {
		t.Fatalf("Get missing: v=%q err=%v", v, err)
	}
	if err := r.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatal(err)
	}
	if v, err := r.Get(ctx, "k"); err != nil || v != "v" {
		t.Fatalf("Get: v=%q err=%v", v, err)
	}
	if ok, err := r.SetNX(ctx, "k", "v2", time.Minute); err != nil || ok {
		t.Fatalf("SetNX existing: ok=%v err=%v", ok, err)
	}
	if n, err := r.Incr(ctx, "n"); err != nil || n != 1 {
		t.Fatalf("Incr: n=%d err=%v", n, err)
	}
	if err := r.Del(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Eval(ctx, "return 1", nil); err == nil {
		t.Fatal("Eval without keys should fail")
	}
	if v, err := r.Eval(ctx, "return nil", []string{"k"}); err != nil || v != nil {
		t.Fatalf("Eval nil reply: v=%v err=%v", v, err)
	}
}

// Get / HGet 对不存在的 key 行为一致；GetOK / HGetOK 能区分"不存在"与"空值"
func TestMissingVsEmpty(t *testing.T) {
	mr := miniredis.RunT(t)
	r := &Redis{Cache: goredis.NewClient(&goredis.Options{Addr: mr.Addr()})}
	ctx := context.Background()

	if v, err := r.HGet(ctx, "h", "f"); err != nil || v != "" {
		t.Fatalf("HGet missing: v=%q err=%v", v, err)
	}
	if _, ok, err := r.GetOK(ctx, "empty"); err != nil || ok {
		t.Fatalf("GetOK missing: ok=%v err=%v", ok, err)
	}
	_ = r.Set(ctx, "empty", "", 0)
	if v, ok, err := r.GetOK(ctx, "empty"); err != nil || !ok || v != "" {
		t.Fatalf("GetOK empty value: v=%q ok=%v err=%v", v, ok, err)
	}
	_ = r.HSet(ctx, "h", "f", "1")
	if v, ok, err := r.HGetOK(ctx, "h", "f"); err != nil || !ok || v != "1" {
		t.Fatalf("HGetOK: v=%q ok=%v err=%v", v, ok, err)
	}
	if _, ok, _ := r.HGetOK(ctx, "h", "nope"); ok {
		t.Fatal("HGetOK missing field must be ok=false")
	}
}

func TestRemainingCommandsAndErrors(t *testing.T) {
	mr := miniredis.RunT(t)
	r := &Redis{Cache: goredis.NewClient(&goredis.Options{Addr: mr.Addr(), MaxRetries: -1})}
	ctx := context.Background()
	if n, _ := r.IncrBy(ctx, "n", 5); n != 5 {
		t.Fatal("incrby")
	}
	if n, _ := r.Decr(ctx, "n"); n != 4 {
		t.Fatal("decr")
	}
	if n, _ := r.DecrBy(ctx, "n", 4); n != 0 {
		t.Fatal("decrby")
	}
	if ok, _ := r.Expire(ctx, "n", time.Minute); !ok {
		t.Fatal("expire")
	}
	if ttl, _ := r.TTL(ctx, "n"); ttl <= 0 {
		t.Fatal("ttl")
	}
	_ = r.HSet(ctx, "h", "a", "1")
	if m, _ := r.HGetAll(ctx, "h"); m["a"] != "1" {
		t.Fatal("hgetall")
	}
	p := r.Pipeline()
	p.Set(ctx, "p", "1", 0)
	if _, err := p.Exec(ctx); err != nil || r.Client() != r.Cache {
		t.Fatal("pipeline/client")
	}
	mr.Close()
	if _, err := r.Get(ctx, "k"); err == nil {
		t.Fatal("Get must return real errors (only redis.Nil is swallowed)")
	}
	if _, _, err := r.HGetOK(ctx, "h", "a"); err == nil {
		t.Fatal("HGetOK must return real errors")
	}
}
