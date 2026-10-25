package core

import (
	"context"
	"fmt"
	"time"
	"willshark/envconfig/config"

	"github.com/redis/go-redis/v9"
)

// NewRedis 按 mode 创建单机 / 集群 / 哨兵客户端并验证连通性。
// 返回的 redis.UniversalClient 在三种模式下用法一致。
func NewRedis(ctx context.Context, conf config.Redis) (redis.UniversalClient, error) {
	var client redis.UniversalClient
	switch conf.Mode {
	case "", config.RedisModeSingle:
		client = redis.NewClient(&redis.Options{
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
	case config.RedisModeCluster:
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           conf.Addrs,
			Password:        conf.Password,
			ReadOnly:        conf.ReadFromReplica,
			RouteRandomly:   conf.ReadFromReplica,
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
	case config.RedisModeSentinel:
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:       conf.MasterName,
			SentinelAddrs:    conf.Addrs,
			SentinelPassword: conf.SentinelPassword,
			ReplicaOnly:      false,
			Password:         conf.Password,
			DB:               conf.DB,
			PoolSize:         conf.PoolSize,
			MinIdleConns:     conf.MinIdleConns,
			MaxIdleConns:     conf.MaxIdleConns,
			ConnMaxIdleTime:  conf.ConnMaxIdleTime,
			DialTimeout:      conf.DialTimeout,
			ReadTimeout:      conf.ReadTimeout,
			WriteTimeout:     conf.WriteTimeout,
			PoolTimeout:      5 * time.Second,
			ConnMaxLifetime:  time.Hour,
			MaxRetries:       3,
			MinRetryBackoff:  8 * time.Millisecond,
			MaxRetryBackoff:  512 * time.Millisecond,
		})
	default:
		return nil, fmt.Errorf("unknown redis mode %q", conf.Mode)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

// PingRedis 健康检查：集群模式检查每个主节点，其余模式 PING
func PingRedis(ctx context.Context, client redis.UniversalClient) error {
	if cc, ok := client.(*redis.ClusterClient); ok {
		return cc.ForEachMaster(ctx, func(ctx context.Context, node *redis.Client) error {
			if err := node.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("node %s: %w", node.Options().Addr, err)
			}
			return nil
		})
	}
	return client.Ping(ctx).Err()
}
