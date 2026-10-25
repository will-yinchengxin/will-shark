package http_client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientV2PostJSON(t *testing.T) {
	type payload struct {
		TenantId     string `json:"tenantId"`
		ProtocolType string `json:"protocol"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/test" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get(AuthHeader); got != "Bearer t" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get(ContentType); got != Json {
			t.Errorf("Content-Type = %q", got)
		}
		var p payload
		_ = json.NewDecoder(r.Body).Decode(&p)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 200, "data": p})
	}))
	defer srv.Close()

	resp, err := NewHttpClient(srv.URL).WithToken("Bearer t").PostJSON("/api/v1/test", payload{TenantId: "t1", ProtocolType: "http"})
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
	if res.Code != 200 || res.Data.TenantId != "t1" {
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
	resp, err := NewHttpClient(srv.URL).Get("/")
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
	resp, err := NewHttpClient(srv.URL).Get("/")
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
	if _, err := PostJsonRequest(`{}`, url); err == nil {
		t.Fatal("expected connection error")
	}
	if _, err := GetJsonRequest(`{}`, url); err == nil {
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

	c := NewHttpClient(srv.URL)
	c.Client.Timeout = 50 * time.Millisecond
	start := time.Now()
	if _, err := c.Get("/"); err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout not applied")
	}
}
