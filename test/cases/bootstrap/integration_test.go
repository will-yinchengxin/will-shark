package bootstrap

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"willshark/app/modules/mysql/dao"
	"willshark/core"
	"willshark/envconfig/config"
)

// TestIntegrationWithDockerCompose 使用真实 MySQL / Redis（docker compose up -d）验证完整启动链路。
// 运行：WS_INTEGRATION=1 go run ./test/run.go ./bootstrap -run Integration
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
	if err := app.Infra.MySQL["willshark"].DB.Raw("SELECT @@character_set_connection").Scan(&charset).Error; err != nil {
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

// 读写分离 / 多主：从库与额外主库指向同一台 docker MySQL，验证节点建连、读写路由、健康检查与关闭
func TestIntegrationMySQLReplicasAndSources(t *testing.T) {
	if os.Getenv("WS_INTEGRATION") == "" {
		t.Skip("set WS_INTEGRATION=1 to run against docker compose")
	}
	t.Setenv(config.EnvConfigDir, "../envconfig")
	cfg, err := config.Load("dev")
	if err != nil {
		t.Fatal(err)
	}
	m := cfg.MySQL["willshark"]
	node := config.MySQLNode{Host: m.Host, Port: m.Port, User: m.User, Password: m.Password, Database: m.Database}
	m.Replicas, m.Sources, m.Policy = []config.MySQLNode{node, node}, []config.MySQLNode{node}, "round_robin"
	conn, err := core.NewMySQL(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	db := conn.DB.WithContext(context.Background())
	if err := db.Exec("CREATE TABLE IF NOT EXISTS it_rw (id INT PRIMARY KEY, v VARCHAR(10))").Error; err != nil {
		t.Fatal(err)
	}
	tableDropped := false
	defer func() {
		// 失败路径也清理；正常路径在 Close 前显式删除，以免在已关闭的连接上执行。
		if !tableDropped {
			_ = db.Exec("DROP TABLE IF EXISTS it_rw").Error
		}
	}()
	if err := db.Exec("REPLACE INTO it_rw VALUES (1, 'x')").Error; err != nil {
		t.Fatal(err)
	}
	var v string
	if err := db.Raw("SELECT v FROM it_rw WHERE id = 1").Scan(&v).Error; err != nil || v != "x" {
		t.Fatalf("read via replica: %q %v", v, err)
	}
	if err := (&dao.MysqlPool{DB: conn.DB}).Master().WithContext(context.Background()).Raw("SELECT v FROM it_rw WHERE id = 1").Scan(&v).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DROP TABLE it_rw").Error; err != nil {
		t.Fatal(err)
	}
	tableDropped = true
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := conn.Ping(context.Background()); err == nil {
		t.Fatal("ping after close must fail")
	}
}
