package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"willshark/app/modules/mysql/dao"
	rc "willshark/app/modules/redis"
	"willshark/core"
	"willshark/envconfig/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func testConfig() *config.Config {
	return &config.Config{
		App: config.App{Name: "test", ID: "1", Env: "test"},
		HTTP: config.HTTP{
			ReadHeaderTimeout: time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
			IdleTimeout: 5 * time.Second, ShutdownTimeout: 5 * time.Second, MaxBodyBytes: 1 << 20,
		},
		Log: config.Log{Level: "info", Format: "json"},
	}
}

// newTestApp 与 New 相同的组装方式，依赖替换为 sqlmock + miniredis
func newTestApp(t *testing.T, cfg *config.Config) (*App, sqlmock.Sqlmock) {
	t.Helper()
	otel.SetTracerProvider(noop.NewTracerProvider())

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := core.RegisterTraceCallbacks(db); err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	cache := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	conn, err := core.WrapGorm(db)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{
		Config: cfg,
		Infra:  &Infra{MySQL: map[string]*core.MySQLConn{"willshark": conn}, Redis: map[string]goredis.UniversalClient{"willshark": cache}},
	}
	routers, sched := wire(cfg, &dao.MysqlPool{DB: db}, &rc.Redis{Cache: cache})
	app.Scheduler = sched
	app.Engine, err = NewEngine(cfg, app.readiness, app.checkDeps, routers)
	if err != nil {
		t.Fatal(err)
	}
	return app, mock
}

type apiResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"message"`
	Data json.RawMessage `json:"data"`
}

func serve(e *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, apiResp) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	var r apiResp
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return w, r
}

func TestAppsEndpoints(t *testing.T) {
	app, mock := newTestApp(t, testConfig())
	e := app.Engine

	mock.ExpectQuery("SELECT \\* FROM `user` WHERE id = \\?").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "will", 18))
	w, r := serve(e, http.MethodPost, "/v1/apps/mysql", `{"name":"a","account":"b"}`)
	if w.Code != http.StatusOK || r.Code != 200 || !strings.Contains(string(r.Data), `"name":"will"`) {
		t.Fatalf("mysql: code=%d resp=%+v data=%s", w.Code, r, r.Data)
	}
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id header missing")
	}

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	if w, r = serve(e, http.MethodPost, "/v1/apps/mysql", `{"name":"a","account":"b"}`); w.Code != http.StatusNotFound || r.Code != 9004 {
		t.Fatalf("not found: code=%d resp=%+v", w.Code, r)
	}

	mock.ExpectQuery("SELECT").WillReturnError(errors.New("connection refused: password=secret"))
	if w, r = serve(e, http.MethodPost, "/v1/apps/mysql", `{"name":"a","account":"b"}`); w.Code != http.StatusInternalServerError || r.Code != 9003 || strings.Contains(r.Msg, "secret") {
		t.Fatalf("db error: code=%d resp=%+v", w.Code, r)
	}

	if w, r = serve(e, http.MethodPost, "/v1/apps/mysql", `{"name":"a"}`); w.Code != http.StatusBadRequest || r.Code != 5003 || !strings.Contains(r.Msg, "account") {
		t.Fatalf("validation: code=%d resp=%+v", w.Code, r)
	}

	if w, r = serve(e, http.MethodPost, "/v1/apps/redis", ""); w.Code != http.StatusOK || string(r.Data) != `{"acquired":true}` {
		t.Fatalf("redis: code=%d resp=%+v", w.Code, r)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppsHTTPRoutesUseController(t *testing.T) {
	app, _ := newTestApp(t, testConfig())
	found := 0
	for _, route := range app.Engine.Routes() {
		if route.Path == "/v1/apps/mysql" || route.Path == "/v1/apps/redis" {
			found++
			if !strings.Contains(route.Handler, "/controller.") {
				t.Errorf("%s %s bypasses controller: %s", route.Method, route.Path, route.Handler)
			}
		}
	}
	if found != 2 {
		t.Fatalf("found %d Apps routes, want 2", found)
	}
}

func TestAppsControllerRejectsInvalidInputBeforeService(t *testing.T) {
	cfg := testConfig()
	cfg.HTTP.MaxBodyBytes = 32
	app, mock := newTestApp(t, cfg)
	for _, tc := range []struct {
		path, body string
		status     int
	}{
		{"/v1/apps/mysql", "{", http.StatusBadRequest},
		{"/v1/apps/mysql", `{"name":"a","account":"b"}{}`, http.StatusBadRequest},
		{"/v1/apps/mysql", `{"name":"a","account":"b"}` + strings.Repeat(" ", 100), http.StatusRequestEntityTooLarge},
		{"/v1/apps/redis", "{", http.StatusBadRequest},
	} {
		w, _ := serve(app.Engine, http.MethodPost, tc.path, tc.body)
		if w.Code != tc.status {
			t.Errorf("%s status=%d body=%s, want %d", tc.path, w.Code, w.Body.String(), tc.status)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEngineInfraRoutes(t *testing.T) {
	cfg := testConfig()
	app, _ := newTestApp(t, cfg)
	e := app.Engine

	if w, _ := serve(e, http.MethodGet, "/healthz", ""); w.Code != http.StatusOK {
		t.Fatalf("healthz=%d", w.Code)
	}
	if w, _ := serve(e, http.MethodGet, "/readyz", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz before start should be 503, got %d", w.Code)
	}
	app.ready.Store(true)
	if w, _ := serve(e, http.MethodGet, "/readyz", ""); w.Code != http.StatusOK {
		t.Fatalf("readyz when ready=%d", w.Code)
	}
	if w, r := serve(e, http.MethodGet, "/nope", ""); w.Code != http.StatusNotFound || r.Code != 404 {
		t.Fatalf("404 envelope: %d %+v", w.Code, r)
	}
	if w, _ := serve(e, http.MethodGet, "/swagger/index.html", ""); w.Code != http.StatusNotFound {
		t.Fatalf("swagger must be off by default, got %d", w.Code)
	}

	cfg2 := testConfig()
	cfg2.App.Swagger = true
	app2, _ := newTestApp(t, cfg2)
	if w, _ := serve(app2.Engine, http.MethodGet, "/swagger/index.html", ""); w.Code != http.StatusOK {
		t.Fatalf("swagger enabled should serve, got %d", w.Code)
	}
}

// 共用依赖故障只体现在 /healthz/deps，/readyz 保持就绪，避免所有 Pod 同时被摘流量
func TestDepsHealthFailsWhenRedisDownButReadyzStaysOK(t *testing.T) {
	app, _ := newTestApp(t, testConfig())
	app.ready.Store(true)
	if w, _ := serve(app.Engine, http.MethodGet, "/healthz/deps", ""); w.Code != http.StatusOK {
		t.Fatalf("deps healthy = %d", w.Code)
	}
	_ = app.Infra.Redis["willshark"].Close()
	if w, _ := serve(app.Engine, http.MethodGet, "/healthz/deps", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("deps with redis down = %d", w.Code)
	}
	if w, _ := serve(app.Engine, http.MethodGet, "/readyz", ""); w.Code != http.StatusOK {
		t.Fatalf("readyz must not depend on shared dependencies, got %d", w.Code)
	}
}

// 优雅退出：收到信号后 /readyz 先变 503（摘流），在途请求正常完成，Serve 正常返回
func TestServeGracefulShutdown(t *testing.T) {
	cfg := testConfig()
	cfg.HTTP.DrainWait = 300 * time.Millisecond
	app, _ := newTestApp(t, cfg)
	started := make(chan struct{})
	app.Engine.GET("/slow", func(c *gin.Context) {
		close(started)
		time.Sleep(500 * time.Millisecond)
		c.String(http.StatusOK, "done")
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() { serveErr <- app.Serve(ctx, ln) }()

	waitFor(t, func() bool {
		resp, err := http.Get(base + "/readyz")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	})

	var wg sync.WaitGroup
	var slowStatus int
	var slowBody string
	wg.Go(func() {
		resp, err := http.Get(base + "/slow")
		if err != nil {
			t.Errorf("in-flight request failed: %v", err)
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		slowStatus, slowBody = resp.StatusCode, string(b)
	})
	<-started
	cancel()

	time.Sleep(100 * time.Millisecond) // 处于 drain 阶段
	resp, err := http.Get(base + "/readyz")
	if err != nil {
		t.Fatalf("server should still accept during drain: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("readyz during drain = %d, want 503", resp.StatusCode)
	}

	wg.Wait()
	if slowStatus != http.StatusOK || slowBody != "done" {
		t.Fatalf("in-flight request: status=%d body=%q", slowStatus, slowBody)
	}
	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	if _, err := http.Get(base + "/healthz"); err == nil {
		t.Fatal("server should be closed after shutdown")
	}
}

func TestClosersReverseOrderAndContinueOnError(t *testing.T) {
	var order []string
	var cl closers
	for _, name := range []string{"tracing", "mysql", "redis"} {
		cl.add(name, func(context.Context) error {
			order = append(order, name)
			if name == "redis" {
				return errors.New("close failed")
			}
			return nil
		})
	}
	cl.closeAll(time.Second)
	if strings.Join(order, ",") != "redis,mysql,tracing" {
		t.Fatalf("order = %v", order)
	}
	cl.closeAll(time.Second) // 再次调用不应重复关闭
	if len(order) != 3 {
		t.Fatalf("closeAll must be idempotent, got %v", order)
	}
}

func TestNewFailsFastAndCleansUp(t *testing.T) {
	cfg := testConfig()
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg.MySQL = map[string]config.MySQL{"willshark": {Host: "127.0.0.1", Port: port, User: "u", Database: "d"}}
	if _, _, err := New(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "init mysql willshark") {
		t.Fatalf("err = %v", err)
	}

	cfg.MySQL = nil
	if _, _, err := New(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "mysql.willshark is not configured") {
		t.Fatalf("err = %v", err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func TestClosersRecoverPanic(t *testing.T) {
	var closed []string
	var cl closers
	cl.add("mysql", func(context.Context) error { closed = append(closed, "mysql"); return nil })
	cl.add("rocketmq", func(context.Context) error { panic("nil pointer") })
	cl.closeAll(time.Second)
	if strings.Join(closed, ",") != "mysql" {
		t.Fatalf("resources after a panicking closer must still be closed: %v", closed)
	}
}

// 在途请求超过 shutdownTimeout 时强制断开，Serve 及时返回（之后才会关闭数据库）
func TestServeForceClosesAfterShutdownTimeout(t *testing.T) {
	cfg := testConfig()
	cfg.HTTP.ShutdownTimeout = 200 * time.Millisecond
	app, _ := newTestApp(t, cfg)
	started, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	app.Engine.GET("/stuck", func(c *gin.Context) {
		close(started)
		select {
		case <-release:
		case <-c.Request.Context().Done(): // 连接被强制关闭时 ctx 取消
		}
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() { serveErr <- app.Serve(ctx, ln) }()
	clientErr := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/stuck")
		if err == nil {
			resp.Body.Close()
		}
		clientErr <- err
	}()
	<-started
	cancel()
	select {
	case err := <-serveErr:
		if err == nil || !strings.Contains(err.Error(), "http shutdown") {
			t.Fatalf("Serve err = %v, want shutdown timeout", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve did not return after shutdown timeout")
	}
	select {
	case err := <-clientErr:
		if err == nil {
			t.Fatal("stuck request should be cut off")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stuck connection was not force-closed")
	}
}

func TestRunListensOnConfiguredPort(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg := testConfig()
	cfg.HTTP.Port = port
	app, _ := newTestApp(t, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	waitFor(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	// 端口被占用时返回 listen 错误
	busy, _ := net.Listen("tcp", ":0")
	defer busy.Close()
	cfg.HTTP.Port = busy.Addr().(*net.TCPAddr).Port
	if err := app.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "listen") {
		t.Fatalf("err = %v", err)
	}
}

// service.Apps.Info：锁被其他请求持有时返回 acquired=false；Redis 不可用时返回 500
func TestAppsRedisLockBranches(t *testing.T) {
	app, _ := newTestApp(t, testConfig())
	cache := app.Infra.Redis["willshark"]
	if err := cache.Set(context.Background(), "client_list", "someone-else", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if w, r := serve(app.Engine, http.MethodPost, "/v1/apps/redis", ""); w.Code != http.StatusOK || string(r.Data) != `{"acquired":false}` {
		t.Fatalf("held lock: %d %+v", w.Code, r)
	}
	_ = cache.Close()
	if w, r := serve(app.Engine, http.MethodPost, "/v1/apps/redis", ""); w.Code != http.StatusInternalServerError || r.Code != 9003 {
		t.Fatalf("redis down: %d %+v", w.Code, r)
	}
}
