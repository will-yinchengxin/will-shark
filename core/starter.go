package core

import (
	"os"
	"runtime"
	"willshark/utils/logs/logger"
)

var (
	StartModule        = make([]func() func(), 0)
	StopModuleFunction = make([]func(), 0)
)

func Start() {
	Environment = "dev"
	if len(os.Args) >= 2 {
		Environment = os.Args[1]
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	//StopModuleFunction = append(StopModuleFunction, initLogger())
	logger.LogToFile()

	// 从本地得yaml中读取配置文件, 可以根据需求调整，如：nacos
	if ok, clearSystemConfig := initCoreConfig(); !ok {
		panic("init system config failed")
	} else {
		StopModuleFunction = append(StopModuleFunction, clearSystemConfig)
	}

	if len(StartModule) > 0 {
		for _, startFunc := range StartModule {
			StopModuleFunction = append(StopModuleFunction, startFunc())
		}
	}
}

func Stop() {
	if len(StopModuleFunction) > 0 {
		for _, clearFunc := range StopModuleFunction {
			clearFunc()
		}
	}
}
