package router

import (
	"github.com/gin-gonic/gin"
	"willshark/app/controller"
)

type ApiRouter struct {
	AppsApi *controller.Apps
}

func (api *ApiRouter) Initialize(app *gin.Engine) {
	v1 := app.Group("v1")
	{
		apps := v1.Group("apps")
		{
			apps.POST("mysql", api.AppsApi.AppList)
			apps.POST("redis", api.AppsApi.AppInfo)
		}
	}
}
