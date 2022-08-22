package cron

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/sirupsen/logrus"
	"will/consts"
	"will/core"
	"will/utils"
	"will/will_tools/logs"
)

func (j *Jobs) mqJob() {
	core.RocketmqConsumerClient.Subscribe(utils.GetMqRealTag(consts.MqTopicChannelReport), consumer.MessageSelector{}, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (c consumer.ConsumeResult, err error) {
		defer func() {
			if err != nil {
				errTag := "mq.subscribe." + consts.MqTopicChannelReport
				_ = core.Log.Error(logs.TraceFormatter{
					Trace: logrus.Fields{
						"msg":  msgs,
						errTag: err.Error(),
					},
				})
			}

		}()

		// Todo trace it with Prometheus
		for i := range msgs {
			select {
			case <-j.Ctx.Done():
				_ = core.Log.Info(logs.TraceFormatter{
					Trace: logrus.Fields{
						"info": "Receiving exit signal of main program，we will stop all the cron jobs",
					},
				})
				return
			default:
				// Todo consume the msg logic ....
				print(i)
			}
		}
		c = consumer.ConsumeSuccess
		return
	})
}
