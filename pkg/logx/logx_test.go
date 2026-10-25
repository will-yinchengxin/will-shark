package logx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
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
	g, _ := m["g"].(map[string]any)
	if g["trace_id"] != tid.String() || g["span_id"] != sid.String() || g["request_id"] != "req-1" {
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
