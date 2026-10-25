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
