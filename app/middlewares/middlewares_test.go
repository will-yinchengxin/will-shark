package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func init() { gin.SetMode(gin.TestMode) }

func corsEngine(origins []string) *gin.Engine {
	e := gin.New()
	e.Use(Cors(origins))
	e.POST("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return e
}

func do(e *gin.Engine, method, origin string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/x", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w
}

func TestCorsWhitelist(t *testing.T) {
	e := corsEngine([]string{"https://ok.com"})

	w := do(e, http.MethodPost, "https://ok.com", nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://ok.com" || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("allowed origin headers wrong: %v", w.Header())
	}

	// 不在白名单：不回写 Origin（旧实现会无条件回写并允许携带 cookie）
	w = do(e, http.MethodPost, "https://evil.com", nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("disallowed origin must not get CORS headers: %v", w.Header())
	}
	w = do(e, http.MethodOptions, "https://evil.com", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("disallowed preflight status = %d", w.Code)
	}

	w = do(e, http.MethodOptions, "https://ok.com", map[string]string{"Access-Control-Request-Headers": "X-Custom"})
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Headers") != "X-Custom" {
		t.Fatalf("preflight: code=%d headers=%v", w.Code, w.Header())
	}

	// 非跨域请求不受影响
	if w = do(e, http.MethodPost, "", nil); w.Code != http.StatusOK || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("same-origin request: code=%d", w.Code)
	}
}

func TestCorsWildcardNoCredentials(t *testing.T) {
	w := do(corsEngine([]string{"*"}), http.MethodPost, "https://any.com", nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("wildcard headers wrong: %v", w.Header())
	}
}

func TestCorsEmptyConfigBlocksAll(t *testing.T) {
	w := do(corsEngine(nil), http.MethodPost, "https://any.com", nil)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("empty whitelist must block: %v", w.Header())
	}
}

func TestStartTraceDisabledPassesContext(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	e := gin.New()
	e.Use(StartTrace())
	called := false
	e.GET("/x", func(c *gin.Context) {
		called = c.Request.Context() != nil
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if !called || w.Code != http.StatusOK {
		t.Fatalf("handler not reached, code=%d", w.Code)
	}
}

// 上游 traceparent 应被继承，handler 中可从 request context 取到同一个 trace
func TestStartTracePropagatesUpstream(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })

	const traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	var inHandler trace.SpanContext
	e := gin.New()
	e.Use(StartTrace())
	e.GET("/x", func(c *gin.Context) {
		inHandler = trace.SpanContextFromContext(c.Request.Context())
		c.Status(http.StatusInternalServerError)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	e.ServeHTTP(httptest.NewRecorder(), req)

	if inHandler.TraceID().String() != traceID {
		t.Fatalf("handler trace id = %s", inHandler.TraceID())
	}
	spans := rec.Ended()
	if len(spans) != 1 || spans[0].SpanContext().TraceID().String() != traceID {
		t.Fatalf("span not continued from upstream: %d spans", len(spans))
	}
	if spans[0].Status().Code.String() != "Error" {
		t.Fatalf("5xx should mark span error, got %v", spans[0].Status())
	}
}
