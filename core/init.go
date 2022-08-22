package core

func init() {
	StartModule = append(StartModule, initMysql)
	StartModule = append(StartModule, initRedis)
	//StartModule = append(StartModule, initRocketMqConfig)
}
