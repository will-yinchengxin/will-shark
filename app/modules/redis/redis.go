// Package redis 对 go-redis UniversalClient 的薄封装，兼容单机 / 集群 / 哨兵。
package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 对 go-redis 的薄封装，所有方法显式接收 ctx（用于超时控制与链路追踪）。
//
// 读取类方法（Get / HGet）在 key 不存在时返回零值与 nil error；
// 需要区分"不存在"与"值为空"时使用 GetOK / HGetOK。
type Redis struct {
	Cache redis.UniversalClient
}

// Get key 不存在时返回 ""、nil
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, _, err := r.GetOK(ctx, key)
	return val, err
}

// GetOK 读取 key；ok=false 表示 key 不存在
func (r *Redis) GetOK(ctx context.Context, key string) (val string, ok bool, err error) {
	return okResult(r.Cache.Get(ctx, key).Result())
}

func okResult(val string, err error) (string, bool, error) {
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Cache.Set(ctx, key, value, expiration).Err()
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	return r.Cache.Del(ctx, keys...).Err()
}

func (r *Redis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.Cache.SetNX(ctx, key, value, expiration).Result()
}

func (r *Redis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return nil, errEmptyKeys
	}

	val, err := r.Cache.Eval(ctx, script, keys, args...).Result()
	if errors.Is(err, redis.Nil) {
		// 脚本返回 nil（如 SET NX 未成功）不是错误
		return nil, nil
	}
	return val, err
}

func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	return r.Cache.Incr(ctx, key).Result()
}

func (r *Redis) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.Cache.IncrBy(ctx, key, value).Result()
}

func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	return r.Cache.Decr(ctx, key).Result()
}

func (r *Redis) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.Cache.DecrBy(ctx, key, value).Result()
}

func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return r.Cache.Expire(ctx, key, expiration).Result()
}

func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Cache.TTL(ctx, key).Result()
}

func (r *Redis) HSet(ctx context.Context, key, field string, value interface{}) error {
	return r.Cache.HSet(ctx, key, field, value).Err()
}

// HGet key 或 field 不存在时返回 ""、nil（与 Get 一致）
func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	val, _, err := r.HGetOK(ctx, key, field)
	return val, err
}

// HGetOK 读取 hash 字段；ok=false 表示 key 或 field 不存在
func (r *Redis) HGetOK(ctx context.Context, key, field string) (val string, ok bool, err error) {
	return okResult(r.Cache.HGet(ctx, key, field).Result())
}

func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Cache.HGetAll(ctx, key).Result()
}

func (r *Redis) Pipeline() redis.Pipeliner {
	return r.Cache.Pipeline()
}

func (r *Redis) Client() redis.UniversalClient {
	return r.Cache
}
