// Package logx 基于标准库 log/slog 的日志初始化。
//
// 业务代码直接使用 slog.InfoContext(ctx, ...) 等函数；传入 ctx 时会自动附带
// trace_id / span_id（开启链路追踪时）与 request_id。
package logx

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

type requestIDKey struct{}

// WithRequestID 把请求 ID 放入 ctx
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID 从 ctx 读取请求 ID
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// New 创建 logger；format 为 json 或 text，attrs 为每条日志的公共字段（如 service、env）
func New(w io.Writer, level, format string, attrs ...any) *slog.Logger {
	opts := &slog.HandlerOptions{Level: ParseLevel(level)}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(contextHandler{h}).With(attrs...)
}

// Setup 创建 logger 并设为全局默认（同时接管标准库 log 包的输出）
func Setup(w io.Writer, level, format string, attrs ...any) *slog.Logger {
	l := New(w, level, format, attrs...)
	slog.SetDefault(l)
	return l
}

func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// contextHandler 从 ctx 中提取 trace/request 信息附加到日志
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
		}
		if id := RequestID(ctx); id != "" {
			r.AddAttrs(slog.String("request_id", id))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
