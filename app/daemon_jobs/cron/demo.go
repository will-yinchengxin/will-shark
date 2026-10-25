// Package cron 业务定时任务。
//
// 每个任务是一个结构体：字段声明它依赖的 service（或 dao），在 bootstrap 中注入；
// Register 声明调度方式，Run 写业务逻辑。任务与 HTTP 接口复用同一套 service，不需要全局变量。
// 新增任务：go tool willshark-gen module <name> -with job
package cron

import (
	"context"
	"log/slog"
	"time"
	"willshark/app/service"
	"willshark/pkg/scheduler"
)

// UserStats 示例：定时统计用户数
type UserStats struct {
	Apps *service.Apps
}

func (j *UserStats) Register(s *scheduler.Scheduler) {
	s.Every("user_stats", 5*time.Minute, j.Run,
		scheduler.WithLock(time.Minute),       // 多副本只执行一次
		scheduler.WithTimeout(30*time.Second), // 单次最长执行时间
	)
}

func (j *UserStats) Run(ctx context.Context) error {
	n, err := j.Apps.CountUsers(ctx)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "user stats", "users", n)
	return nil
}
