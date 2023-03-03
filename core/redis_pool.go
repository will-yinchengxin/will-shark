package core

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"time"
)

var (
	RedisConfig map[string]interface{}
	RedisPool   map[string]*redis.Pool
)

type RedisConn struct {
	Host           string `yaml:"host,omitempty"`
	PassWord       string `yaml:"password,omitempty"`
	DataBase       int    `yaml:"database,omitempty"`
	MaxIdleNum     int    `yaml:"maxIdleNum,omitempty"`
	MaxActive      int    `yaml:"maxActive,omitempty"`
	MaxIdleTimeout int    `yaml:"maxIdleTimeout,omitempty"`
	ConnectTimeout int    `yaml:"connectTimeout,omitempty"`
	ReadTimeout    int    `yaml:"readTimeout,omitempty"`
}

func initRedis() func() {
	RedisConfig, err := GetMapConfig(CoreConfig, "redis", RedisConn{})
	if err != nil {
		return func() {
		}
	}
	if len(RedisConfig) < 1 {
		panic("init redis pool config failed, redis config not found")
	}
	RedisPool = make(map[string]*redis.Pool)
	for name, val := range RedisConfig {
		if rdb, _ := initRedisPool(val.(*RedisConn)); rdb != nil {
			RedisPool[name] = rdb
		}
	}

	return func() {
		for key, value := range RedisPool {
			if err := value.Close(); err != nil {
				_ = Log.PanicDefault("Redis[ " + key + " ]Close Err:" + err.Error())
				continue
			}
			_ = Log.SuccessDefault("Redis[ " + key + " ]Close Success!")
		}

	}
}

func initRedisPool(conne *RedisConn) (*redis.Pool, error) {
	redisPool := &redis.Pool{
		MaxIdle:     conne.MaxIdleNum,
		MaxActive:   conne.MaxActive,
		IdleTimeout: time.Duration(conne.MaxIdleTimeout) * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			con, err := redis.Dial("tcp", conne.Host,
				redis.DialPassword(conne.PassWord), redis.DialDatabase(conne.DataBase),
				redis.DialConnectTimeout(time.Duration(conne.ConnectTimeout)*time.Second),
				redis.DialReadTimeout(time.Duration(conne.ReadTimeout)*time.Second),
				redis.DialWriteTimeout(time.Duration(conne.ReadTimeout)*time.Second),
			)
			if err != nil {
				return nil, err
			}
			return con, nil
		},
	}
	return redisPool, nil
}

func GetRedisDB(key string) (rdb *redis.Pool, err error) {
	rdbPool, ok := RedisPool[key]
	if !ok {
		if config, ok := RedisConfig[key]; !ok {
			return nil, errors.New(key + " redis dbConfig doesn't exist")
		} else {
			if rdbPool, err := initRedisPool(config.(*RedisConn)); err != nil {
				return nil, errors.New(key + " redis dbConfig Initialization failure")
			} else {
				RedisPool[key] = rdbPool
			}
		}
	}
	return rdbPool, nil
}
