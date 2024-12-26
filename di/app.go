package di

import (
	"willshark/app/controller"
	"willshark/app/daemon_jobs/cron"
	"willshark/app/modules/mysql/dao"
	"willshark/app/router"
	"willshark/app/service"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

var Injector = wire.NewSet(wire.Struct(new(App), "*"))

type App struct {
	Engine *gin.Engine
	Jobs   *cron.Jobs
}

var CronJobSet = wire.NewSet(
	wire.Struct(new(cron.Jobs), "*"),
)

var RouterSet = wire.NewSet(
	wire.Struct(new(router.Routers), "*"),
	wire.Struct(new(router.ApiRouter), "*"),
)

var ControllerSet = wire.NewSet(
	wire.Struct(new(controller.Apps), "*"),
)

var ServiceSet = wire.NewSet(
	wire.Struct(new(service.Apps), "*"),
)

var Daoset = wire.NewSet(
	wire.Struct(new(dao.User), "*"),
)

var MiddlewaresSet = wire.NewSet()
