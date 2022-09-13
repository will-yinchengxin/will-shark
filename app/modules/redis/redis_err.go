package redis

import "errors"

var (
	getKeyErr     = errors.New("Redis Get err : ")
	setKeyErr     = errors.New("Redis Set err : ")
	expireErr     = errors.New("Redis Expire err : ")
	delErr        = errors.New("Redis Del err : ")
	setWitLockErr = errors.New("Redis SetWitLock err : ")
	lenKeyErr     = errors.New("len key is zero")
	evalCtxErr    = errors.New("Redis EvalCtx err : ")
)
