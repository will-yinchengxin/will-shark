package middlewares

import (
	"context"
	"github.com/gin-gonic/gin"
	"willshark/consts"
	"willshark/core"
)

func StartTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		if core.Trace != nil {
			tr := core.Trace.Tracer(consts.JaegerRouter)
			ctxRoot, span := tr.Start(context.Background(), c.Request.URL.Path)
			c.Set(consts.JaegerContext, ctxRoot)
			defer span.End()
			c.Next()
		}
	}
}
