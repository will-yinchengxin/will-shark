package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestClientV2PostJSON(t *testing.T) {
	type payload struct {
		TenantID     string `json:"tenantId"`
		ProtocolType string `json:"protocol"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/test" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get(AuthHeader); got != "Bearer t" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get(ContentType); got != JSON {
			t.Errorf("Content-Type = %q", got)
		}
		var p payload
		_ = json.NewDecoder(r.Body).Decode(&p)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": p})
	}))
	defer srv.Close()

	resp, err := NewHTTPClient(srv.URL).WithToken("Bearer t").PostJSON("/api/v1/test", payload{TenantID: "t1", ProtocolType: "http"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := HandleResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Code int     `json:"code"`
		Data payload `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || res.Data.TenantID != "t1" {
		t.Fatalf("unexpected response %+v", res)
	}
}

func TestClientV2NoTokenNoAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(AuthHeader) != "" {
			t.Errorf("Authorization header should not be sent without token")
		}
	}))
	defer srv.Close()
	resp, err := NewHTTPClient(srv.URL).Get("/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HandleResponse(resp); err != nil {
		t.Fatal(err)
	}
}

func TestHandleResponseNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	resp, err := NewHTTPClient(srv.URL).Get("/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HandleResponse(resp); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

// 连接失败时旧实现会在 defer resp.Body.Close() 处 nil panic
func TestLegacyClientConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	if _, err := PostJSONRequest(`{}`, url); err == nil {
		t.Fatal("expected connection error")
	}
	if _, err := GetJSONRequest(`{}`, url); err == nil {
		t.Fatal("expected connection error")
	}
}

func TestClientTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)

	c := NewHTTPClient(srv.URL)
	c.Client.Timeout = 50 * time.Millisecond
	start := time.Now()
	if _, err := c.Get("/"); err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout not applied")
	}
}

func TestTracePropagatedAnd2xxAccepted(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	var traceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	resp, err := NewHTTPClient(srv.URL).PostJSONCtx(ctx, "/x", map[string]int{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	body, err := HandleResponse(resp)
	if err != nil || string(body) != `{"id":1}` {
		t.Fatalf("201 must be treated as success: body=%s err=%v", body, err)
	}
	if !strings.Contains(traceparent, sc.TraceID().String()) {
		t.Fatalf("traceparent not propagated: %q", traceparent)
	}
}

func TestPutDeleteAndRequestSize(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL)
	for _, do := range []func() (*http.Response, error){
		func() (*http.Response, error) { return c.PutJSON("/x", map[string]int{"a": 1}) },
		func() (*http.Response, error) { return c.Delete("/x") },
	} {
		resp, err := do()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := HandleResponse(resp); err != nil {
			t.Fatalf("204 must be success: %v", err)
		}
	}
	if strings.Join(methods, ",") != "PUT,DELETE" {
		t.Fatalf("methods = %v", methods)
	}
	if _, err := c.PostJSON("/x", make(chan int)); err == nil {
		t.Fatal("unmarshalable body must fail")
	}
	req := httptest.NewRequest(http.MethodPost, "/abc", strings.NewReader("12345"))
	req.Header.Set("K", "V")
	if got := ComputeAppRequestSize(req); got != len("/abc")+len("POST")+len("HTTP/1.1")+len("K")+len("V")+len("example.com")+5 {
		t.Fatalf("size = %d", got)
	}
}

func TestHandleResponseTooLarge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, MaxResponseBytes+1))
	}))
	defer srv.Close()
	resp, err := NewHTTPClient(srv.URL).Get("/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HandleResponse(resp); err == nil {
		t.Fatal("oversized body must be rejected")
	}
}
