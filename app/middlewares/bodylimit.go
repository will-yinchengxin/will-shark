package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit 限制请求体大小，超出时读取 body 会返回 *http.MaxBytesError（httpx 映射为 413）
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
