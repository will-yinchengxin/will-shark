package cron

import (
	"context"
	"github.com/sirupsen/logrus"
	"time"
	"will/core"
	"will/will_tools/logs"
)

// register your own cron job

type Jobs struct {
	Ctx       context.Context
	CancelFun context.CancelFunc
}

func (j *Jobs) RegisterJobs() func() {
	defer func() {
		if err := recover(); err != nil {
			logInfo := logs.TraceFormatter{
				Trace: logrus.Fields{
					"error": err,
				},
			}
			_ = core.Log.Panic(logInfo)
		}
	}()

	go j.startNormal()
	//go j.startMQ()

	return j.logOutJobs
}

func (j *Jobs) logOutJobs() {
	j.CancelFun()
	time.Sleep(time.Second * 2)
	logInfo := logs.TraceFormatter{
		Trace: logrus.Fields{
			"info": "\n End scheduled task \n",
		},
	}
	_ = core.Log.Info(logInfo)
}

func (j *Jobs) startMQ() {
	var err error
	defer func() {
		if err != nil {
			errTag := "mq.start"
			_ = core.Log.Error(logs.TraceFormatter{
				Trace: logrus.Fields{
					errTag: err.Error(),
				},
			})
		}
	}()
	j.mqJob()
}

func (j *Jobs) startNormal() {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			jobOne(t)
		case <-j.Ctx.Done():
			_ = core.Log.Info(logs.TraceFormatter{
				Trace: logrus.Fields{
					"info": "Receiving exit signal of main program，we will stop all the cron jobs",
				},
			})
			return
		}
	}
}
