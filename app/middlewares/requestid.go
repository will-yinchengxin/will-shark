// Package middlewares gin 中间件：请求 ID、链路追踪、访问日志、panic 恢复、请求体大小限制、CORS。
package middlewares

import (
	"crypto/rand"
	"encoding/hex"
	"willshark/pkg/logx"

	"github.com/gin-gonic/gin"
)

const HeaderRequestID = "X-Request-Id"

// RequestID 透传或生成请求 ID，写入响应头与 request context（日志自动带 request_id）
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Header(HeaderRequestID, id)
		c.Request = c.Request.WithContext(logx.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// validRequestID 只接受长度合理的可见 ASCII，防止日志注入
func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < 0x21 || id[i] > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
