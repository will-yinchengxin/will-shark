package redis

import (
	"context"
	"time"
	"willshark/utils/logs/logger"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	Cache *redis.Client
	Ctx   context.Context
}

func (r *Redis) WithContext(ctx context.Context) *Redis {
	return &Redis{
		Cache: r.Cache,
		Ctx:   ctx,
	}
}

func (r *Redis) Get(key string) (string, error) {
	val, err := r.Cache.Get(r.Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		logger.Error(getKeyErr.Error() + err.Error())
		return "", err
	}
	return val, nil
}

func (r *Redis) Set(key string, value interface{}, expiration time.Duration) error {
	err := r.Cache.Set(r.Ctx, key, value, expiration).Err()
	if err != nil {
		logger.Error(setKeyErr.Error() + err.Error())
		return err
	}
	return nil
}

func (r *Redis) Del(keys ...string) error {
	err := r.Cache.Del(r.Ctx, keys...).Err()
	if err != nil {
		logger.Error(delErr.Error() + err.Error())
		return err
	}
	return nil
}

func (r *Redis) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := r.Cache.SetNX(r.Ctx, key, value, expiration).Result()
	if err != nil {
		logger.Error(setWitLockErr.Error() + err.Error())
		return false, err
	}
	return ok, nil
}

func (r *Redis) Eval(script string, keys []string, args ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return nil, lenKeyErr
	}

	val, err := r.Cache.Eval(r.Ctx, script, keys, args...).Result()
	if err != nil {
		logger.Error(evalCtxErr.Error() + err.Error())
		return nil, err
	}
	return val, nil
}

func (r *Redis) Incr(key string) (int64, error) {
	return r.Cache.Incr(r.Ctx, key).Result()
}

func (r *Redis) IncrBy(key string, value int64) (int64, error) {
	return r.Cache.IncrBy(r.Ctx, key, value).Result()
}

func (r *Redis) Decr(key string) (int64, error) {
	return r.Cache.Decr(r.Ctx, key).Result()
}

func (r *Redis) DecrBy(key string, value int64) (int64, error) {
	return r.Cache.DecrBy(r.Ctx, key, value).Result()
}

func (r *Redis) Expire(key string, expiration time.Duration) (bool, error) {
	return r.Cache.Expire(r.Ctx, key, expiration).Result()
}

func (r *Redis) TTL(key string) (time.Duration, error) {
	return r.Cache.TTL(r.Ctx, key).Result()
}

func (r *Redis) HSet(key, field string, value interface{}) error {
	return r.Cache.HSet(r.Ctx, key, field, value).Err()
}

func (r *Redis) HGet(key, field string) (string, error) {
	return r.Cache.HGet(r.Ctx, key, field).Result()
}

func (r *Redis) HGetAll(key string) (map[string]string, error) {
	return r.Cache.HGetAll(r.Ctx, key).Result()
}

func (r *Redis) Pipeline() redis.Pipeliner {
	return r.Cache.Pipeline()
}

func (r *Redis) Client() *redis.Client {
	return r.Cache
}
