package core

import (
	"context"
	"errors"
	"sync/atomic"
	"willshark/envconfig/config"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// RocketMQ 生产者已启动；消费者在 Consumer.Subscribe 之后调用 StartConsumer 启动
type RocketMQ struct {
	Producer rocketmq.Producer
	Consumer rocketmq.PushConsumer

	consumerStarted atomic.Bool
}

// StartConsumer 启动消费者（需先 Subscribe）。请使用本方法而不是 Consumer.Start，
// 以便 Close 时正确关闭：未启动的 PushConsumer 调用 Shutdown 会 panic。
func (m *RocketMQ) StartConsumer() error {
	if err := m.Consumer.Start(); err != nil {
		return err
	}
	m.consumerStarted.Store(true)
	return nil
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
		_ = p.Shutdown()
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
	var errs []error
	if m.consumerStarted.Load() {
		errs = append(errs, m.Consumer.Shutdown())
	}
	return errors.Join(append(errs, m.Producer.Shutdown())...)
}
