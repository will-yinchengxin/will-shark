package controller

import (
	"willshark/app/do/request"
	"willshark/app/service"
	"willshark/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// Apps 管理示例接口的 HTTP 输入与输出。
type Apps struct {
	Svc *service.Apps
}

// List 从 MySQL 获取用户信息。
//
//	@Summary	获取App信息
//	@Description	通过 mysql 获取信息
//	@Tags		App相关
//	@Accept		json
//	@Produce	json
//	@Param		body	body	request.Apps	true	"请求参数"
//	@Success	200	{object}	httpx.Response{data=response.User}
//	@Router		/v1/apps/mysql [post]
func (c *Apps) List(ctx *gin.Context) {
	var req request.Apps
	if err := httpx.Bind(ctx, &req); err != nil {
		httpx.Fail(ctx, err)
		return
	}
	resp, err := c.Svc.List(ctx.Request.Context(), &req)
	if err != nil {
		httpx.Fail(ctx, err)
		return
	}
	if resp == nil {
		httpx.OK(ctx, nil)
		return
	}
	httpx.OK(ctx, resp)
}

// Info 获取 Redis 分布式锁示例结果。
//
//	@Summary	获取App信息
//	@Description	通过 redis 获取信息
//	@Tags		App相关
//	@Produce	json
//	@Success	200	{object}	httpx.Response{data=response.LockInfo}
//	@Router		/v1/apps/redis [post]
func (c *Apps) Info(ctx *gin.Context) {
	var req request.LockInfo
	if err := httpx.Bind(ctx, &req); err != nil {
		httpx.Fail(ctx, err)
		return
	}
	resp, err := c.Svc.Info(ctx.Request.Context(), &req)
	if err != nil {
		httpx.Fail(ctx, err)
		return
	}
	if resp == nil {
		httpx.OK(ctx, nil)
		return
	}
	httpx.OK(ctx, resp)
}
