package cron

import (
	"context"
	"time"
	"willshark/app/modules/mysql/dao"
	conn "willshark/app/modules/redis"
	"willshark/utils/logs/logger"
)

type Jobs struct {
	Rds  *conn.Redis
	User dao.User
}

func (j *Jobs) RegisterJobs() func() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err.(error).Error())
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
			logger.Info("Receiving Exit Signal Of Main Program，Stop All The CronJobs Success")
			return
		case <-time.Tick(time.Duration(300-time.Now().Second()) * time.Second):
			j.oneCronJob()
		}
	}
}
