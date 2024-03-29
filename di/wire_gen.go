// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//+build !wireinject

package di

import (
	"will/app/controller"
	"will/app/daemon_jobs/cron"
	"will/app/modules/mysql/dao"
	"will/app/router"
	"will/app/service"
	"will/utils/validator"
)

// Injectors from wire.go:

func BuildInjector() (*App, func(), error) {
	validatorX := validator.NewValidator()
	redisPool, cleanup := InitRedis()
	mysqlPool, cleanup2 := InitMysql()
	user := dao.User{
		MysqlPool: mysqlPool,
	}
	apps := service.Apps{
		Rds:  redisPool,
		User: user,
	}
	controllerApps := &controller.Apps{
		RequestValidate: validatorX,
		App:             apps,
	}
	apiRouter := &router.ApiRouter{
		AppsApi: controllerApps,
	}
	routers := &router.Routers{
		Api: apiRouter,
	}
	engine, cleanup3 := InitGinEngine(routers)
	jobs := &cron.Jobs{
		Rds:  redisPool,
		User: user,
	}
	app := &App{
		Engine: engine,
		Jobs:   jobs,
	}
	return app, func() {
		cleanup3()
		cleanup2()
		cleanup()
	}, nil
}
