package core

import (
	"context"
	"errors"
	"time"
	"willshark/utils/logs/logger"

	"github.com/redis/go-redis/v9"
)

var (
	RedisConfig map[string]interface{}
	RedisPool   map[string]*redis.Client
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
		return func() {}
	}
	if len(RedisConfig) < 1 {
		panic("init redis pool config failed, redis config not found")
	}
	RedisPool = make(map[string]*redis.Client)
	for name, val := range RedisConfig {
		rdb, err := initRedisClient(val.(*RedisConn))
		if err != nil {
			panic("initRedisClient err: " + err.Error())
		}
		if rdb != nil {
			RedisPool[name] = rdb
		}
	}

	return func() {
		for key, client := range RedisPool {
			if err := client.Close(); err != nil {
				logger.Error("Redis[ " + key + " ]Close Err:" + err.Error())
				continue
			}
			logger.Info("Redis[ " + key + " ]Close Success!")
		}
	}
}

func initRedisClient(conf *RedisConn) (*redis.Client, error) {
	options := &redis.Options{
		Addr:            conf.Host,
		Password:        conf.PassWord,
		DB:              conf.DataBase,
		MaxIdleConns:    conf.MaxIdleNum,
		PoolSize:        conf.MaxActive,
		ConnMaxIdleTime: time.Duration(conf.MaxIdleTimeout) * time.Second,
		DialTimeout:     time.Duration(conf.ConnectTimeout) * time.Second,
		ReadTimeout:     time.Duration(conf.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(conf.ReadTimeout) * time.Second,

		// 自定义配置
		MinIdleConns:    conf.MaxIdleNum / 4,
		PoolTimeout:     5 * time.Second,
		ConnMaxLifetime: time.Hour,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	}

	client := redis.NewClient(options)

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

func GetRedisDB(key string) (*redis.Client, error) {
	client, ok := RedisPool[key]
	if !ok {
		config, exists := RedisConfig[key]
		if !exists {
			return nil, errors.New(key + " redis dbConfig doesn't exist")
		}

		newClient, err := initRedisClient(config.(*RedisConn))
		if err != nil {
			return nil, errors.New(key + " redis dbConfig Initialization failure")
		}
		RedisPool[key] = newClient
		return newClient, nil
	}
	return client, nil
}
