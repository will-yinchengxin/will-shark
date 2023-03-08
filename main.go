package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"will/core"
	"will/di"
)

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
		_ = core.Log.SuccessDefault(fmt.Sprintf("Receive System signal: %s", sig))
		core.Stop()
		return
	}
}
