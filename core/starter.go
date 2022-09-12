package core

import (
	"os"
	"fmt"
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
	
	fmt.Printf("runtime.NumCPU(): %v\n", runtime.NumCPU())
    	runtime.GOMAXPROCS(runtime.NumCPU())

	StopModuleFunction = append(StopModuleFunction, initLogger())

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
