package middlewares

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"willshark/pkg/logx"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(logx.New(&buf, "debug", "json"))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("bad log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func TestRequestID(t *testing.T) {
	e := gin.New()
	e.Use(RequestID())
	var inCtx string
	e.GET("/x", func(c *gin.Context) { inCtx = logx.RequestID(c.Request.Context()) })

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	gen := w.Header().Get(HeaderRequestID)
	if len(gen) != 32 || inCtx != gen {
		t.Fatalf("generated id=%q ctx=%q", gen, inCtx)
	}

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "upstream-123")
	w = httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Header().Get(HeaderRequestID) != "upstream-123" || inCtx != "upstream-123" {
		t.Fatal("upstream request id should be passed through")
	}

	// 含换行等不可见字符的 ID 会被替换，防止日志注入
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(HeaderRequestID, "evil id")
	w = httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Header().Get(HeaderRequestID) == "evil id" {
		t.Fatal("invalid request id should be replaced")
	}
}

func TestAccessLogRedactsAndOmitsHeaders(t *testing.T) {
	buf := captureLogs(t)
	e := gin.New()
	e.Use(RequestID(), AccessLog(true))
	var seenBody string
	e.POST("/login", func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		seenBody = string(b)
		c.Status(http.StatusUnauthorized)
	})

	body := `{"user":"will","password":"p@ss","profile":{"phone":"13800138000"},"items":[{"token":"t"}]}`
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer SECRET-TOKEN")
	req.Header.Set("Cookie", "sid=SECRET-COOKIE")
	e.ServeHTTP(httptest.NewRecorder(), req)

	if seenBody != body {
		t.Fatalf("handler must still receive the full body, got %q", seenBody)
	}
	raw := buf.String()
	for _, secret := range []string{"p@ss", "13800138000", "SECRET-TOKEN", "SECRET-COOKIE", `"t"`} {
		if strings.Contains(raw, secret) {
			t.Fatalf("log leaks %q: %s", secret, raw)
		}
	}
	lines := logLines(t, buf)
	if len(lines) != 1 {
		t.Fatalf("want 1 access log, got %d", len(lines))
	}
	l := lines[0]
	if l["msg"] != "access" || l["status"] != float64(401) || l["level"] != "WARN" || l["route"] != "/login" || l["request_id"] == nil {
		t.Fatalf("access log fields wrong: %v", l)
	}
	if !strings.Contains(l["req_body"].(string), `"user":"will"`) {
		t.Fatalf("non-sensitive fields should be kept: %v", l["req_body"])
	}
}

func TestAccessLogSkipsLargeBody(t *testing.T) {
	buf := captureLogs(t)
	e := gin.New()
	e.Use(AccessLog(true))
	e.POST("/x", func(c *gin.Context) {})
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"a":"`+strings.Repeat("x", 4096)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(httptest.NewRecorder(), req)
	l := logLines(t, buf)[0]
	if _, ok := l["req_body"]; ok {
		t.Fatal("large body must not be logged")
	}
	if l["req_bytes"] == nil {
		t.Fatal("req_bytes should be logged")
	}
}

func TestRecovery(t *testing.T) {
	buf := captureLogs(t)
	e := gin.New()
	e.Use(AccessLog(false), Recovery())
	e.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
	var r struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	if r.Code != 9003 || strings.Contains(r.Msg, "kaboom") {
		t.Fatalf("response must be generic: %+v", r)
	}
	var panicLog, accessLog map[string]any
	for _, l := range logLines(t, buf) {
		switch l["msg"] {
		case "panic recovered":
			panicLog = l
		case "access":
			accessLog = l
		}
	}
	if panicLog == nil || !strings.Contains(panicLog["stack"].(string), "recovery_test") && !strings.Contains(panicLog["stack"].(string), "observability_test") {
		t.Fatalf("panic log with stack missing: %v", panicLog)
	}
	if accessLog == nil || accessLog["status"] != float64(500) {
		t.Fatalf("access log should record 500: %v", accessLog)
	}
}

func TestBodyLimit(t *testing.T) {
	e := gin.New()
	e.Use(BodyLimit(4))
	var readErr error
	e.POST("/x", func(c *gin.Context) { _, readErr = io.ReadAll(c.Request.Body) })
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("123456")))
	var mbe *http.MaxBytesError
	if readErr == nil || !errorsAs(readErr, &mbe) {
		t.Fatalf("expected MaxBytesError, got %v", readErr)
	}
}

func TestRedactJSONNonJSON(t *testing.T) {
	if RedactJSON([]byte("password=1")) != "<non-json body>" {
		t.Fatal("non-json must not be echoed")
	}
}

func errorsAs(err error, target any) bool { return errors.As(err, target) }

func TestStartTraceUsesRouteTemplate(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	e := gin.New()
	e.Use(StartTrace())
	e.GET("/items/:id", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items/42", nil))

	spans := rec.Ended()
	if len(spans) != 1 || spans[0].Name() != "GET /items/:id" || spans[0].Status().Code != codes.Error {
		t.Fatalf("spans = %+v", spans)
	}
}

func TestCorsExposesRequestID(t *testing.T) {
	e := gin.New()
	e.Use(Cors([]string{"https://ok.com"}))
	e.GET("/x", func(c *gin.Context) {})
	for origin, want := range map[string]string{"https://ok.com": HeaderRequestID, "https://evil.com": ""} {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, req)
		if got := w.Header().Get("Access-Control-Expose-Headers"); got != want {
			t.Errorf("origin %s: expose = %q, want %q", origin, got, want)
		}
	}
}

func TestRecoveryBrokenPipeAndAbortHandler(t *testing.T) {
	brokenPipe := &net.OpError{Op: "write", Err: &os.SyscallError{Syscall: "write", Err: syscall.EPIPE}}
	if !isBrokenPipe(brokenPipe) || isBrokenPipe(errors.New("x")) {
		t.Fatal("isBrokenPipe")
	}
	buf := captureLogs(t)
	e := gin.New()
	e.Use(Recovery())
	e.GET("/pipe", func(*gin.Context) { panic(brokenPipe) })
	e.GET("/abort", func(*gin.Context) { panic(http.ErrAbortHandler) })

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/pipe", nil))
	if strings.Contains(buf.String(), "panic recovered") {
		t.Fatal("broken pipe must not be logged as a panic")
	}
	func() {
		defer func() {
			if r := recover(); r != http.ErrAbortHandler {
				t.Fatalf("ErrAbortHandler must be re-panicked for net/http, got %v", r)
			}
		}()
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/abort", nil))
	}()
}
