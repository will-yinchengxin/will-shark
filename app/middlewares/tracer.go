package middlewares

import (
	"willshark/consts"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// StartTrace 从请求头提取上游 trace（W3C traceparent），并把 span 放入 c.Request.Context()
// 未开启链路追踪时全局 tracer 为 noop，开销可忽略
func StartTrace() gin.HandlerFunc {
	tr := otel.Tracer(consts.JaegerRouter)
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		// span 名用路由模板（/v1/order/:id）而不是实际路径，避免每个 ID 一个 span 名导致链路系统高基数
		name, route := c.Request.Method, c.FullPath()
		if route != "" {
			name += " " + route
		}
		ctx, span := tr.Start(ctx, name, trace.WithSpanKind(trace.SpanKindServer), trace.WithAttributes(
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.String("url.path", c.Request.URL.Path),
		))
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Next()

		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.response.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, "")
		}
	}
}
