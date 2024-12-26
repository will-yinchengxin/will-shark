package lock

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
	"willshark/app/modules/redis"
	"willshark/utils/logs/logger"
	"willshark/utils/strings"
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewRedisLock(store *redis.Redis, key string) *RedisLock {
	return &RedisLock{
		store: store,
		key:   key,
		id:    strings.Randn(randomLen),
	}
}

func (rl *RedisLock) SetExpire(seconds int) {
	atomic.StoreUint32(&rl.seconds, uint32(seconds))
}

func (rl *RedisLock) Acquire() (bool, error) {
	return rl.AcquireCtx(context.Background())
}

func (rl *RedisLock) AcquireCtx(ctx context.Context) (bool, error) {
	seconds := atomic.LoadUint32(&rl.seconds)
	resp, err := rl.store.Eval(lockCommand,
		[]string{rl.key}, rl.id, strconv.Itoa(int(seconds)*millisPerSecond+tolerance))

	if err != nil {
		// Todo: when do lock_test annotation the log plugin
		logger.Error(fmt.Sprintf("Error on acquiring lock for %s, %s", rl.key, err.Error()))
		return false, err
	} else if resp == nil {
		return false, nil
	}

	reply, ok := resp.(string)
	if ok && reply == "OK" {
		return true, nil
	}

	// Todo: when do lock_test annotation the log plugin
	logger.Error(fmt.Sprintf("Unknown reply when acquiring lock for %s: %v", rl.key, resp))

	return false, nil
}

func (rl *RedisLock) Release() (bool, error) {
	return rl.ReleaseCtx(context.Background())
}

func (rl *RedisLock) ReleaseCtx(ctx context.Context) (bool, error) {
	resp, err := rl.store.Eval(delCommand, []string{rl.key}, rl.id)
	if err != nil {
		logger.Error(redis.ReleaseCtxErr.Error() + err.Error())

		return false, err
	}

	reply, ok := resp.(int64)
	if !ok {
		return false, nil
	}

	return reply == 1, nil
}
