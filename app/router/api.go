package router

import (
	"willshark/app/controller"

	"github.com/gin-gonic/gin"
)

type APIRouter struct {
	Apps *controller.Apps
}

func (api *APIRouter) Initialize(engine *gin.Engine) {
	v1 := engine.Group("v1")
	{
		apps := v1.Group("apps")
		{
			apps.POST("mysql", api.Apps.List)
			apps.POST("redis", api.Apps.Info)
		}
	}
}
