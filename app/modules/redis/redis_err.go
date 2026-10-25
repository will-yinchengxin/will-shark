package redis

import "errors"

var errEmptyKeys = errors.New("redis: eval requires at least one key")
