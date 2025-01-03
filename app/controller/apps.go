package controller

import (
	"context"
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
	var (
		req           request.Apps
		spanFatherCtx context.Context
	)
	if errMsg := apps.RequestValidate.ParseJson(ctx, &req); errMsg != "" {
		utils.MessageError(ctx, errMsg)
		return
	}

	spanFather, _ := ctx.Get("traceRoot")
	spanFatherCtx = spanFather.(context.Context)

	res, codeType := apps.App.List(req, spanFatherCtx)
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
	var (
		spanFatherCtx context.Context
	)
	spanFatherCtx = context.Background()
	res, codeType := apps.App.Info(spanFatherCtx)
	if codeType.Code != 0 {
		utils.Error(ctx, codeType)
		return
	}
	utils.Out(ctx, res)
	return
}
