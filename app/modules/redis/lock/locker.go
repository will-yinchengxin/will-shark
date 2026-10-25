package lock

import (
	"context"
	"log/slog"
	"math"
	"time"
	"willshark/app/modules/redis"
)

// Locker 基于 RedisLock 的分布式锁，满足 scheduler.Locker 接口
type Locker struct {
	Store *redis.Redis
}

func NewLocker(store *redis.Redis) *Locker { return &Locker{Store: store} }

// TryLock 非阻塞获取锁；ttl 向上取整到秒
func (l *Locker) TryLock(ctx context.Context, key string, ttl time.Duration) (func(context.Context), bool, error) {
	rl := NewRedisLock(l.Store, key)
	rl.SetExpire(int(math.Ceil(ttl.Seconds())))
	ok, err := rl.AcquireCtx(ctx)
	if err != nil || !ok {
		return nil, ok, err
	}
	return func(ctx context.Context) {
		if _, err := rl.ReleaseCtx(ctx); err != nil {
			slog.WarnContext(ctx, "release lock failed", "key", key, "err", err)
		}
	}, true, nil
}
