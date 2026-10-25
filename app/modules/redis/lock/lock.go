// Package lock 基于 Redis 的分布式锁，Locker 可直接用于 scheduler.WithLocker。
package lock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"willshark/app/modules/redis"
)

const (
	randomLen       = 16
	tolerance       = 500 // milliseconds
	millisPerSecond = 1000
)

type RedisLock struct {
	store   *redis.Redis
	seconds uint32
	key     string
	id      string
}

func NewRedisLock(store *redis.Redis, key string) *RedisLock {
	return &RedisLock{
		store: store,
		key:   key,
		id:    randomID(),
	}
}

// randomID 锁持有者标识，使用加密随机数，避免不同进程生成相同 ID 而误删他人的锁
func randomID() string {
	b := make([]byte, randomLen/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SetExpire 锁的过期秒数（实际 TTL 额外加 500ms 容差）；未设置或 <=0 时只有 500ms
func (rl *RedisLock) SetExpire(seconds int) {
	if seconds < 0 {
		seconds = 0
	}
	atomic.StoreUint32(&rl.seconds, uint32(seconds))
}

func (rl *RedisLock) Acquire() (bool, error) {
	return rl.AcquireCtx(context.Background())
}

func (rl *RedisLock) AcquireCtx(ctx context.Context) (bool, error) {
	seconds := atomic.LoadUint32(&rl.seconds)
	resp, err := rl.store.Eval(ctx, lockCommand,
		[]string{rl.key}, rl.id, strconv.Itoa(int(seconds)*millisPerSecond+tolerance))

	if err != nil {
		return false, fmt.Errorf("acquire lock %s: %w", rl.key, err)
	} else if resp == nil {
		return false, nil
	}

	reply, ok := resp.(string)
	if ok && reply == "OK" {
		return true, nil
	}

	slog.WarnContext(ctx, "unknown reply when acquiring lock", "key", rl.key, "reply", resp)

	return false, nil
}

func (rl *RedisLock) Release() (bool, error) {
	return rl.ReleaseCtx(context.Background())
}

func (rl *RedisLock) ReleaseCtx(ctx context.Context) (bool, error) {
	resp, err := rl.store.Eval(ctx, delCommand, []string{rl.key}, rl.id)
	if err != nil {
		return false, fmt.Errorf("release lock %s: %w", rl.key, err)
	}

	reply, ok := resp.(int64)
	if !ok {
		return false, nil
	}

	return reply == 1, nil
}
