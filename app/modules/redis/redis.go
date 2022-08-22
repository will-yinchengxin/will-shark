package redis

import (
	"github.com/gomodule/redigo/redis"
	"will/core"
	"will/will_tools/logs"
)

type RedisPool struct {
	DB   *redis.Pool
	Conn redis.Conn
}

func (r *RedisPool) Get(key string) string {
	Val, err := redis.String(r.Conn.Do("get", key))
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return ""
	}
	return Val
}

func (r *RedisPool) Set(key string, val int) bool {
	_, err := redis.String(r.Conn.Do("set", key, val))
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return false
	}
	return true
}

func (r *RedisPool) Expire(key string) {
	_, err := r.Conn.Do("expire", key, 10)
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return
	}
}

func (r *RedisPool) Del(key string) {
	_, err := r.Conn.Do("del", key)
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return
	}
}
