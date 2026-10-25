package router

import "github.com/gin-gonic/gin"

type RouterInterface interface {
	Initialize(engine *gin.Engine)
}

// Routers 汇总所有路由组；由 bootstrap 组装。
// 注释 willshark:gen:* 为代码生成器的插入点，请保留。
type Routers struct {
	Api *ApiRouter
	// willshark:gen:router-field
}

func (r *Routers) SetupRouter(engine *gin.Engine) *gin.Engine {
	r.Api.Initialize(engine)
	v1 := engine.Group("/v1")
	_ = v1
	// willshark:gen:router-register
	return engine
}
