package router

import "github.com/gin-gonic/gin"

type RouterInterface interface {
	Initialize(engine *gin.Engine)
}

type Routers struct {
	Api *ApiRouter
}

func (r *Routers) SetupRouter(engine *gin.Engine) *gin.Engine {
	r.Api.Initialize(engine)
	return engine
}
