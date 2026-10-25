package cron

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"
	"willshark/app/modules/mysql/dao"
	conn "willshark/app/modules/redis"
)

// Jobs 进程内定时任务。
// 注意：多副本部署时每个副本都会执行；需要全局只执行一次的任务请加分布式锁（见 redis/lock）。
type Jobs struct {
	Rds  *conn.Redis
	User dao.User

	wg sync.WaitGroup
}

// Start 启动所有任务，ctx 取消后任务在当前一轮执行完后退出
func (j *Jobs) Start(ctx context.Context) {
	j.every(ctx, "job_one", 5*time.Minute, j.oneCronJob)
}

// Wait 等待所有任务退出，超过 ctx 期限则放弃等待
func (j *Jobs) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		j.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("all cron jobs stopped")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait cron jobs: %w", ctx.Err())
	}
}

func (j *Jobs) every(ctx context.Context, name string, interval time.Duration, fn func(context.Context)) {
	j.wg.Go(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				j.runSafely(ctx, name, fn)
			}
		}
	})
}

func (j *Jobs) runSafely(ctx context.Context, name string, fn func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "cron job panic", "job", name, "err", r, "stack", string(debug.Stack()))
		}
	}()
	fn(ctx)
}
