package redis

import "errors"

var (
	lenKeyErr     = errors.New("len key is zero")
	AcquireCtxErr = errors.New("Redis Lock AcquireCtx err : ")
	ReleaseCtxErr = errors.New("Redis Lock ReleaseCtx err : ")
)
