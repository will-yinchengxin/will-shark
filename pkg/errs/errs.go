// Package errs 业务错误：携带对外错误码、HTTP 状态码、对外提示，以及不对外暴露的底层原因。
//
//	return nil, errs.NotFound                       // 直接返回预定义错误
//	return nil, errs.Internal.Wrap(err)             // 包装底层错误，对外只显示通用提示
//	return nil, errs.InvalidParam.WithMsg("手机号格式错误")
//	errors.Is(err, errs.NotFound)                   // 按错误码判断
package errs

import (
	"context"
	"errors"
	"net/http"
)

type Error struct {
	Code   int    // 对外业务码（响应体 code）
	Status int    // HTTP 状态码
	Msg    string // 对外提示
	cause  error  // 底层原因，只进日志，不返回给前端
}

// New 定义一个业务错误；建议在包级别以变量形式定义并复用
func New(code, status int, msg string) *Error {
	return &Error{Code: code, Status: status, Msg: msg}
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Msg + ": " + e.cause.Error()
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.cause }

// Is 按错误码比较，使 errors.Is(errs.NotFound.Wrap(x), errs.NotFound) 为 true
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code
}

// Wrap 返回附带底层原因的副本，不修改原错误
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// WithMsg 返回替换对外提示的副本
func (e *Error) WithMsg(msg string) *Error {
	c := *e
	c.Msg = msg
	return &c
}

// From 将任意 error 转为 *Error；非 *Error 视为内部错误。
// 底层原因是 ctx 取消（客户端断开）或超时的 5xx 错误单独归类为 Canceled / Timeout，
// 例如 errs.Internal.Wrap(ctxErr)，避免把客户端断开记成服务端故障。
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) && e.Status < http.StatusInternalServerError {
		return e
	}
	switch {
	case errors.Is(err, context.Canceled):
		return Canceled.Wrap(err)
	case errors.Is(err, context.DeadlineExceeded):
		return Timeout.Wrap(err)
	case e != nil:
		return e
	}
	return Internal.Wrap(err)
}

// 预定义错误；错误码沿用框架原有取值，避免前端改动
var (
	InvalidParam    = New(5003, http.StatusBadRequest, "参数错误")
	Unauthorized    = New(4010, http.StatusUnauthorized, "未登录或登录已过期")
	Forbidden       = New(4030, http.StatusForbidden, "无权限")
	NotFound        = New(9004, http.StatusNotFound, "资源不存在")
	Conflict        = New(4090, http.StatusConflict, "资源冲突")
	TooLarge        = New(4130, http.StatusRequestEntityTooLarge, "请求体过大")
	TooManyRequests = New(4290, http.StatusTooManyRequests, "请求过于频繁，请稍后重试")
	Canceled        = New(4990, 499, "请求已取消") // 客户端断开（nginx 约定的 499）
	Timeout         = New(9005, http.StatusGatewayTimeout, "请求超时，请稍后重试")
	Failure         = New(9001, http.StatusInternalServerError, "请求失败")
	Upstream        = New(9002, http.StatusBadGateway, "请求依赖的服务失败")
	Internal        = New(9003, http.StatusInternalServerError, "服务器异常，请稍后重试")
)
