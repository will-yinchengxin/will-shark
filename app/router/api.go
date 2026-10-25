package router

import (
	"willshark/app/service"
	"willshark/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type APIRouter struct {
	Apps *service.Apps
}

func (api *APIRouter) Initialize(engine *gin.Engine) {
	v1 := engine.Group("v1")
	{
		apps := v1.Group("apps")
		{
			apps.POST("mysql", httpx.Handle(api.Apps.List))
			apps.POST("redis", httpx.Handle(api.Apps.Info))
		}
	}
}
