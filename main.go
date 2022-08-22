package main

import (
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
	core.StopModuleFunction = append(core.StopModuleFunction, app.CronJobs.RegisterJobs())

	// start server
	core.StopModuleFunction = append(core.StopModuleFunction, core.HttpStarter(app.Engine))

	defer func() { core.Stop() }()
}
