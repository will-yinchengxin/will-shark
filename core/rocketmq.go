package core

import (
	"context"
	"errors"
	"willshark/envconfig/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// RocketMQ 生产者已启动；消费者需在 Subscribe 之后由调用方 Start
type RocketMQ struct {
	Producer rocketmq.Producer
	Consumer rocketmq.PushConsumer
}

func NewRocketMQ(conf config.RocketMQ) (*RocketMQ, error) {
	rlog.SetLogLevel("error")
	p, err := rocketmq.NewProducer(
		producer.WithNameServer(conf.NameServers),
		producer.WithRetry(conf.Retry),
		producer.WithGroupName(conf.GroupName),
	)
	if err != nil {
		return nil, err
	}
	if err := p.Start(); err != nil {
		return nil, err
	}
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(conf.NameServers),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(conf.GroupName),
	)
	if err != nil {
		_ = p.Shutdown()
		return nil, err
	}
	return &RocketMQ{Producer: p, Consumer: c}, nil
}

func (m *RocketMQ) Close(context.Context) error {
	return errors.Join(m.Consumer.Shutdown(), m.Producer.Shutdown())
}
