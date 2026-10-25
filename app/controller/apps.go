package controller

import (
	"github.com/gin-gonic/gin"
	"willshark/app/do/request"
	"willshark/app/service"
	"willshark/utils"
	"willshark/utils/validator"
)

type Apps struct {
	RequestValidate *validator.ValidatorX
	App             service.Apps
}

func (apps *Apps) AppList(ctx *gin.Context) {
	var req request.Apps
	if errMsg := apps.RequestValidate.ParseJson(ctx, &req); errMsg != "" {
		utils.MessageError(ctx, errMsg)
		return
	}

	// 链路追踪中间件已把 span 放入 request context
	res, codeType := apps.App.List(req, ctx.Request.Context())
	if codeType.Code != 0 {
		utils.Error(ctx, codeType)
		return
	}
	utils.Out(ctx, res)
	return
}

// @Summary 获取App信息
// @Description 通过 redis 获取信息
// @Tags App相关
// @ID AppInfo
// @Accept json
// @Produce json
// @Router /v1/apps/redis [post]
func (apps *Apps) AppInfo(ctx *gin.Context) {
	res, codeType := apps.App.Info(ctx.Request.Context())
	if codeType.Code != 0 {
		utils.Error(ctx, codeType)
		return
	}
	utils.Out(ctx, res)
	return
}
