package bootstrap

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"willshark/envconfig/config"
)

// TestIntegrationWithDockerCompose 使用真实 MySQL / Redis（docker compose up -d）验证完整启动链路。
// 运行：WS_INTEGRATION=1 go test ./bootstrap -run Integration
func TestIntegrationWithDockerCompose(t *testing.T) {
	if os.Getenv("WS_INTEGRATION") == "" {
		t.Skip("set WS_INTEGRATION=1 to run against docker compose")
	}
	t.Setenv(config.EnvConfigDir, "../envconfig")
	cfg, err := config.Load("dev")
	if err != nil {
		t.Fatal(err)
	}
	app, cleanup, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	var charset string
	if err := app.Infra.DB["willshark"].Raw("SELECT @@character_set_connection").Scan(&charset).Error; err != nil {
		t.Fatal(err)
	}
	if charset != "utf8mb4" {
		t.Fatalf("connection charset = %s, want utf8mb4", charset)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Serve(ctx, ln) }()
	base := "http://" + ln.Addr().String()
	waitFor(t, func() bool {
		resp, err := http.Get(base + "/readyz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	})

	resp, err := http.Post(base+"/v1/apps/mysql", "application/json", strings.NewReader(`{"name":"a","account":"b"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mysql endpoint status = %d", resp.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
