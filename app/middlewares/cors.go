package middlewares

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

const defaultAllowHeaders = "Authorization, Content-Type, X-Token, X-Request-Id"

// Cors 按白名单放行跨域请求
//   - allowOrigins 包含 "*"：放行任意来源，但不允许携带凭证（cookie）
//   - 其他情况：仅回写白名单内的 Origin，并允许携带凭证
//   - 不在白名单内：不写任何 CORS 头，由浏览器拦截
func Cors(allowOrigins []string) gin.HandlerFunc {
	allowAll := slices.Contains(allowOrigins, "*")
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		h := c.Writer.Header()
		h.Add("Vary", "Origin")
		switch {
		case allowAll:
			h.Set("Access-Control-Allow-Origin", "*")
		case slices.Contains(allowOrigins, origin):
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
		default:
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		if c.Request.Method == http.MethodOptions {
			allowHeaders := c.GetHeader("Access-Control-Request-Headers")
			if allowHeaders == "" {
				allowHeaders = defaultAllowHeaders
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", allowHeaders)
			h.Set("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
