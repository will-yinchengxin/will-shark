package di

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"will/app/controller"
	"will/app/daemon_jobs/cron"
	"will/app/modules/mysql/dao"
	"will/app/router"
	"will/app/service"
)

var Injector = wire.NewSet(wire.Struct(new(App), "*"))

type App struct {
	Engine   *gin.Engine
	CronJobs *cron.Jobs
}

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
