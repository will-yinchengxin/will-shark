package middlewares

import (
	"context"
	"github.com/gin-gonic/gin"
	"will/consts"
	"will/core"
)

func StartTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		tr := core.Trace.Tracer(consts.JaegerRouter)
		ctxRoot, span := tr.Start(context.Background(), c.Request.URL.Path)
		c.Set(consts.JaegerContext, ctxRoot)
		defer span.End()
		c.Next()
	}
}
