package redis

import (
	"context"
	"errors"
	"time"
	"willshark/utils/logs/logger"

	"github.com/redis/go-redis/v9"
)

// Redis 对 go-redis 的薄封装，所有方法显式接收 ctx（用于超时控制与链路追踪）
type Redis struct {
	Cache *redis.Client
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	val, err := r.Cache.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		logger.Error(getKeyErr.Error() + err.Error())
		return "", err
	}
	return val, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	err := r.Cache.Set(ctx, key, value, expiration).Err()
	if err != nil {
		logger.Error(setKeyErr.Error() + err.Error())
		return err
	}
	return nil
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	err := r.Cache.Del(ctx, keys...).Err()
	if err != nil {
		logger.Error(delErr.Error() + err.Error())
		return err
	}
	return nil
}

func (r *Redis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := r.Cache.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		logger.Error(setWitLockErr.Error() + err.Error())
		return false, err
	}
	return ok, nil
}

func (r *Redis) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return nil, lenKeyErr
	}

	val, err := r.Cache.Eval(ctx, script, keys, args...).Result()
	if errors.Is(err, redis.Nil) {
		// 脚本返回 nil（如 SET NX 未成功）不是错误
		return nil, nil
	}
	if err != nil {
		logger.Error(evalCtxErr.Error() + err.Error())
		return nil, err
	}
	return val, nil
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

func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Cache.HGet(ctx, key, field).Result()
}

func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Cache.HGetAll(ctx, key).Result()
}

func (r *Redis) Pipeline() redis.Pipeliner {
	return r.Cache.Pipeline()
}

func (r *Redis) Client() *redis.Client {
	return r.Cache
}
