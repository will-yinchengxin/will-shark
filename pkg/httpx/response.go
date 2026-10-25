// Package httpx HTTP 层的统一约定：参数绑定 + 校验、统一响应结构、错误到响应的映射。
package httpx

import (
	"log/slog"
	"net/http"
	"sync/atomic"
	"willshark/pkg/errs"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK = 200
	MsgOK  = "请求成功"
)

// Response 统一响应结构：{"code":200,"message":"请求成功","data":{...}}
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
	Data any    `json:"data"`
}

var alwaysOK atomic.Bool

// SetAlwaysOK 为 true 时业务错误也返回 HTTP 200，仅靠 body.code 区分（兼容旧前端）
func SetAlwaysOK(v bool) { alwaysOK.Store(v) }

// OK 输出成功响应；data 为 nil 时输出 {}
func OK(c *gin.Context, data any) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, Response{Code: CodeOK, Msg: MsgOK, Data: data})
}

// Fail 输出错误响应并中止后续 handler。
// 5xx 错误记录 error 日志（含底层原因）；对外只返回 errs.Error 的 Msg，不暴露内部细节。
func Fail(c *gin.Context, err error) {
	e := errs.From(err)
	if e.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request.Context(), "request failed", "code", e.Code, "err", err)
	}
	Abort(c, err)
}

// Abort 输出错误响应但不记录日志（调用方已自行记录时使用）
func Abort(c *gin.Context, err error) {
	e := errs.From(err)
	_ = c.Error(err)

	status := e.Status
	if alwaysOK.Load() {
		status = http.StatusOK
	}
	c.AbortWithStatusJSON(status, Response{Code: e.Code, Msg: e.Msg, Data: struct{}{}})
}
