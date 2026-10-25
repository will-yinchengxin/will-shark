package core

import (
	"context"
	"time"
	"willshark/envconfig/config"

	"github.com/redis/go-redis/v9"
)

// NewRedis 创建 Redis 客户端并验证连通性
func NewRedis(ctx context.Context, conf config.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            conf.Addr,
		Password:        conf.Password,
		DB:              conf.DB,
		PoolSize:        conf.PoolSize,
		MinIdleConns:    conf.MinIdleConns,
		MaxIdleConns:    conf.MaxIdleConns,
		ConnMaxIdleTime: conf.ConnMaxIdleTime,
		DialTimeout:     conf.DialTimeout,
		ReadTimeout:     conf.ReadTimeout,
		WriteTimeout:    conf.WriteTimeout,
		PoolTimeout:     5 * time.Second,
		ConnMaxLifetime: time.Hour,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
