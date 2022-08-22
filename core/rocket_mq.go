package core

import (
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

var (
	RocketmqProducerClient rocketmq.Producer
	RocketmqConsumerClient rocketmq.PushConsumer
)

type RocketmqConfig struct {
	Host      []string          `yaml:"host,omitempty"`
	Retry     int               `yaml:"retry,omitempty"`
	GroupName string            `yaml:"groupName,omitempty"`
	Topic     map[string]string `yaml:"topic,omitempty"`
}

var RocketConfig *RocketmqConfig

func initRocketMqConfig() func() {
	cfg, err := GetSingleConfig(CoreConfig, "rocketmq", RocketmqConfig{})
	if err != nil {
		panic("init rocketmq failed, rocketmq config not found")
	}
	rlog.SetLogLevel("error")
	var ok bool
	RocketConfig, ok = cfg.(*RocketmqConfig)
	if !ok {
		panic("init rocketmq failed, rocketmq config incorrect")
	}
	RocketmqProducerClient, err = rocketmq.NewProducer(
		producer.WithNameServer(RocketConfig.Host),
		producer.WithRetry(RocketConfig.Retry),
		producer.WithGroupName(RocketConfig.GroupName),
	)

	if err != nil {
		panic(err)
	}
	err = RocketmqProducerClient.Start()
	if err != nil {
		panic(err)
	}
	RocketmqConsumerClient, err = rocketmq.NewPushConsumer(
		consumer.WithNameServer(RocketConfig.Host),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(RocketConfig.GroupName),
	)
	if err != nil {
		panic(err)
	}
	return func() {
		RocketmqProducerClient.Shutdown()
		RocketmqConsumerClient.Shutdown()
	}
}
