package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// maxLoggedBody 访问日志中记录的请求体上限；超过则只记录大小
const maxLoggedBody = 2 << 10

// sensitiveKeys 日志脱敏字段（小写，精确匹配 JSON 键名）
var sensitiveKeys = map[string]struct{}{
	"password": {}, "passwd": {}, "pwd": {}, "old_password": {}, "new_password": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "accesstoken": {}, "refreshtoken": {},
	"secret": {}, "secret_key": {}, "secretkey": {}, "api_key": {}, "apikey": {},
	"authorization": {}, "cookie": {},
	"phone": {}, "mobile": {}, "id_card": {}, "idcard": {}, "bank_card": {}, "bankcard": {},
}

// AccessLog 每个请求一条结构化日志。
// 不记录请求头（Authorization/Cookie 等），logRequestBody 开启时记录脱敏后的 JSON 请求体（≤2KB）。
func AccessLog(logRequestBody bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var body []byte
		if logRequestBody && c.ContentType() == binding.MIMEJSON && c.Request.ContentLength > 0 && c.Request.ContentLength <= maxLoggedBody {
			body, _ = io.ReadAll(io.LimitReader(c.Request.Body, maxLoggedBody))
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), c.Request.Body))
		}

		c.Next()

		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = "NOT_FOUND"
		}
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("resp_bytes", c.Writer.Size()),
			slog.String("user_agent", c.Request.UserAgent()),
		}
		if c.Request.ContentLength > 0 {
			attrs = append(attrs, slog.Int64("req_bytes", c.Request.ContentLength))
		}
		if len(body) > 0 {
			attrs = append(attrs, slog.String("req_body", RedactJSON(body)))
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.Last().Error()))
		}

		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		slog.LogAttrs(c.Request.Context(), level, "access", attrs...)
	}
}

// RedactJSON 对 JSON 中的敏感字段打码；非法 JSON 不输出原文
func RedactJSON(raw []byte) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "<non-json body>"
	}
	out, _ := json.Marshal(redact(v))
	return string(out)
}

func redact(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
				t[k] = "***"
				continue
			}
			t[k] = redact(val)
		}
	case []any:
		for i := range t {
			t[i] = redact(t[i])
		}
	}
	return v
}
