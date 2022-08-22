//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"will/utils/validator"
)

func BuildInjector() (*App, func(), error) {
	wire.Build(
		ControllerSet,
		ServiceSet,
		RouterSet,
		Injector,
		Daoset,
		InitCronJobs,
		InitGinEngine,
		InitMysql,
		InitRedis,
		validator.NewValidator,
	)
	return new(App), func() {}, nil
}
