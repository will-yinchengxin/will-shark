package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // 内置时区数据，容器内无需安装 tzdata
	"willshark/core"
	"willshark/di"
	"willshark/utils/logs/logger"
)

// version 构建时通过 -ldflags "-X main.version=..." 注入
var version = "dev"

func main() {
	core.Start()
	app, clearDi, err := di.BuildInjector()
	if err != nil || app == nil || app.Engine == nil {
		panic("init di failed")
	}
	core.StopModuleFunction = append(core.StopModuleFunction, clearDi)
	// daemon job
	core.StopModuleFunction = append(core.StopModuleFunction, app.Jobs.RegisterJobs())

	// listen to the stop signal
	logger.Info("willshark version: " + version)
	go listenToSystemSignals()

	// start server
	httpServer, httpServerCancelFunc := core.HttpStarter(app.Engine)
	core.StopModuleFunction = append(core.StopModuleFunction, httpServerCancelFunc)
	httpServer.ListenAndServe()
}

func listenToSystemSignals() {
	signalChan := make(chan os.Signal, 1)
	signals := []os.Signal{syscall.SIGKILL, syscall.SIGSTOP,
		syscall.SIGINT, syscall.SIGQUIT, syscall.SIGILL,
		syscall.SIGABRT, syscall.SIGSYS, syscall.SIGTERM}
	signal.Notify(signalChan, signals...)

	for {
		sig := <-signalChan
		logger.Info(fmt.Sprintf("Receive System signal: %s", sig))
		core.Stop()
		return
	}
}
