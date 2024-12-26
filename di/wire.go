//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"willshark/utils/validator"
)

func BuildInjector() (*App, func(), error) {
	wire.Build(
		ControllerSet,
		ServiceSet,
		RouterSet,
		Injector,
		Daoset,
		CronJobSet,
		InitGinEngine,
		InitMysql,
		InitRedis,
		validator.NewValidator,
	)
	return new(App), func() {}, nil
}
