package redis

import (
	"context"
	"github.com/gomodule/redigo/redis"
	"will/core"
	"will/will_tools/logs"
)

type RedisPool struct {
	Cache *redis.Pool
	Conn  redis.Conn
	Ctx   context.Context
}

func (r *RedisPool) WithContext(ctx context.Context) *RedisPool {
	r.Ctx = ctx
	return r
}

func (r *RedisPool) Get(key string) string {
	Val, err := redis.String(r.Cache.Get().Do("get", key))
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: getKeyErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return ""
	}
	return Val
}

func (r *RedisPool) Set(key string, val int) bool {
	_, err := redis.String(r.Cache.Get().Do("set", key, val))
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: setKeyErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return false
	}
	return true
}

func (r *RedisPool) Expire(key string) bool {
	_, err := r.Cache.Get().Do("expire", key, 10)
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: expireErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return false
	}
	return true
}

func (r *RedisPool) Del(key string) bool {
	_, err := r.Cache.Get().Do("del", key)
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: delErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return false
	}
	return true
}

func (r *RedisPool) SetWitLock(key string, val string, time int) bool { //SET test 1 EX 10 NX
	intVal, err := r.Cache.Get().Do("set", key, val, "EX", time, "NX")
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: setWitLockErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return false
	}
	if intVal != nil {
		return true
	}
	return false
}

func (r *RedisPool) Eval(script string, keys []string, args ...interface{}) (interface{}, error) {
	return r.EvalCtx(context.Background(), script, keys, args)
}

func (r *RedisPool) EvalCtx(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if len(keys) == 0 {
		return nil, lenKeyErr
	}
	cmdArgs := make([]interface{}, 0, len(keys)+len(args))
	for _, val := range keys {
		cmdArgs = append(cmdArgs, val)
	}
	cmdArgs = append(cmdArgs, args...)
	finalCmdArgs := r.args(script, len(keys), cmdArgs)
	val, err := r.Cache.Get().Do("EVAL", finalCmdArgs...)
	if err != nil {
		logInfo := logs.StringFormatter{
			Msg: evalCtxErr.Error() + err.Error(),
		}
		_ = core.Log.Error(logInfo)
		return nil, err
	}
	return val, err
}

func (r *RedisPool) args(spec string, keyCount int, keysAndArgs []interface{}) []interface{} {
	var args []interface{}
	if keyCount <= 0 {
		args = make([]interface{}, 1+len(keysAndArgs))
		args[0] = spec
		copy(args[1:], keysAndArgs)
	} else {
		args = make([]interface{}, 2+len(keysAndArgs))
		args[0] = spec
		args[1] = keyCount
		copy(args[2:], keysAndArgs)
	}
	return args
}
