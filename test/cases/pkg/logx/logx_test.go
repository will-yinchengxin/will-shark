package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestContextFieldsAndCommonAttrs(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info", "json", "service", "svc")

	tid, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	sid, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid}))
	ctx = WithRequestID(ctx, "req-1")

	l.With("k", "v").WithGroup("g").InfoContext(ctx, "hello", "x", 1)

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("%v: %s", err, buf.String())
	}
	if m["service"] != "svc" || m["k"] != "v" || m["msg"] != "hello" {
		t.Fatalf("common attrs missing: %v", m)
	}
	// 上下文字段在顶层（不随 WithGroup 嵌套），分组内只有业务属性
	g, _ := m["g"].(map[string]any)
	if m["trace_id"] != tid.String() || m["span_id"] != sid.String() || m["request_id"] != "req-1" || g["x"] != float64(1) {
		t.Fatalf("context fields missing: %v", m)
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "warn", "text")
	l.Info("hidden")
	if buf.Len() != 0 {
		t.Fatalf("info should be filtered at warn level: %s", buf.String())
	}
	l.Warn("shown")
	if buf.Len() == 0 {
		t.Fatal("warn should be logged")
	}
}

func TestParseLevel(t *testing.T) {
	if ParseLevel("DEBUG") != slog.LevelDebug || ParseLevel("nonsense") != slog.LevelInfo {
		t.Fatal("ParseLevel mapping wrong")
	}
}

func TestNoContextIsSafe(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, "info", "json").Info("no ctx")
	if buf.Len() == 0 {
		t.Fatal("expected output")
	}
}

// WithGroup 之后 request_id / trace_id 仍在顶层，分组内的属性仍在分组内
func TestContextFieldsStayTopLevelWithGroup(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info", "json", "service", "s")
	ctx := WithRequestID(context.Background(), "rid-1")
	l.With("a", 1).WithGroup("db").With("table", "user").WithGroup("q").InfoContext(ctx, "m", "sql", "x")

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	db, _ := m["db"].(map[string]any)
	q, _ := db["q"].(map[string]any)
	if m["request_id"] != "rid-1" || m["service"] != "s" || m["a"] != float64(1) ||
		db["table"] != "user" || q["sql"] != "x" || db["request_id"] != nil {
		t.Fatalf("unexpected layout: %s", buf.String())
	}
}

func TestSetupSetsDefaultAndEmptyGroup(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	var buf bytes.Buffer
	l := Setup(&buf, "debug", "text", "service", "s")
	if slog.Default() != l {
		t.Fatal("Setup must set the default logger")
	}
	slog.Default().WithGroup("").DebugContext(WithRequestID(context.Background(), "r"), "m")
	if out := buf.String(); !strings.Contains(out, "request_id=r") || !strings.Contains(out, "service=s") {
		t.Fatalf("out = %s", out)
	}
	if ParseLevel("ERROR") != slog.LevelError || ParseLevel("x") != slog.LevelInfo {
		t.Fatal("ParseLevel")
	}
}
