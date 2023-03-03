package cron

import (
	"context"
	"time"
	"will/app/modules/mysql/dao"
	conn "will/app/modules/redis"
	"will/core"
)

type Jobs struct {
	Rds  *conn.RedisPool
	User dao.User
}

func (j *Jobs) RegisterJobs() func() {
	defer func() {
		if err := recover(); err != nil {
			_ = core.Log.PanicDefault(err.(error).Error())
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	go j.startCronJobs(ctx)
	return func() {
		cancel()
	}
}

func (j *Jobs) startCronJobs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = core.Log.SuccessDefault("Receiving Exit Signal Of Main Program，Stop All The CronJobs Success")
			return
		case <-time.Tick(time.Duration(300-time.Now().Second()) * time.Second):
			j.oneCronJob()
		}
	}
}
