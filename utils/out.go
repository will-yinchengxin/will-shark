package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	SUCCESS                    = &CodeType{200, "请求成功"}
	FAILURE                    = &CodeType{9001, "请求失败"}
	RequestOtherServiceFailure = &CodeType{9002, "请求依赖的服务失败"}
	GlobalError                = &CodeType{9003, "服务器异常,请稍后重试"}
	NotFoundError              = &CodeType{9004, "NOT FOUND"}

	DBError = &CodeType{1000, "DB Err"}
)

type CodeType struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

type RepType struct {
	CodeType
	Data interface{} `json:"data"`
}

func SendResponse(rep *CodeType, data interface{}) *RepType {
	r := new(RepType)
	r.Code = rep.Code
	r.Msg = rep.Msg
	r.Data = data
	return r
}

// 统一返回响应接口格式
func Out(ctx *gin.Context, data interface{}) {
	retData := SendResponse(SUCCESS, data)
	ctx.JSON(http.StatusOK, retData)
	ctx.Abort()
	return
}

// 请求成功
func Success(ctx *gin.Context) {
	retData := SendResponse(SUCCESS, map[string]interface{}{})
	ctx.JSON(http.StatusOK, retData)
	ctx.Abort()
	return
}

// Error 输出错误
func Error(ctx *gin.Context, rep *CodeType) {
	retData := SendResponse(rep, map[string]interface{}{})
	ctx.JSON(http.StatusOK, retData)
	ctx.Abort()
	return
}

// Error 输出错误并携带信息
func ErrorWithData(ctx *gin.Context, rep *CodeType, data interface{}) {
	retData := SendResponse(rep, data)
	ctx.JSON(http.StatusOK, retData)
	ctx.Abort()
	return
}

// 错误信息返回
func MessageError(ctx *gin.Context, msg string) {
	retData := SendResponse(&CodeType{Code: 5003, Msg: msg}, map[string]interface{}{})
	ctx.JSON(http.StatusOK, retData)
	ctx.Abort()
	return
}
