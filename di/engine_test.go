package di

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"willshark/app/controller"
	"willshark/app/modules/mysql/dao"
	rc "willshark/app/modules/redis"
	"willshark/app/router"
	"willshark/app/service"
	"willshark/core"
	"willshark/utils/validator"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 按 wire_gen.go 相同的方式组装 engine，依赖替换为 sqlmock + miniredis，链路追踪关闭
func newTestEngine(t *testing.T) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	otel.SetTracerProvider(noop.NewTracerProvider())
	core.CoreConfig = map[string]interface{}{}

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
	rds := &rc.Redis{Cache: goredis.NewClient(&goredis.Options{Addr: mr.Addr()})}

	user := dao.User{MysqlPool: &dao.MysqlPool{DB: db}}
	apps := service.Apps{Rds: rds, User: user}
	routers := &router.Routers{Api: &router.ApiRouter{AppsApi: &controller.Apps{
		RequestValidate: validator.NewValidator(),
		App:             apps,
	}}}
	engine, _ := InitGinEngine(routers)
	return engine, mock
}

type resp struct {
	Code int             `json:"code"`
	Msg  string          `json:"message"`
	Data json.RawMessage `json:"data"`
}

func post(t *testing.T, e *gin.Engine, path, body string) (int, resp) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	var r resp
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return w.Code, r
}

// 回归：链路追踪关闭时 /v1/apps/mysql 曾因 traceRoot 断言与 gorm 回调 panic
func TestAppsMysqlEndpoint(t *testing.T) {
	e, mock := newTestEngine(t)
	mock.ExpectQuery("SELECT \\* FROM `user` WHERE id = \\?").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "age"}).AddRow(1, "will", 18))

	code, r := post(t, e, "/v1/apps/mysql", `{"name":"a","account":"b"}`)
	if code != http.StatusOK || r.Code != 200 {
		t.Fatalf("code=%d resp=%+v", code, r)
	}
	if !strings.Contains(string(r.Data), `"name":"will"`) {
		t.Fatalf("data=%s", r.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAppsMysqlEndpointNotFound(t *testing.T) {
	e, mock := newTestEngine(t)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, r := post(t, e, "/v1/apps/mysql", `{"name":"a","account":"b"}`)
	if r.Code != 9004 {
		t.Fatalf("want NotFound(9004), got %+v", r)
	}
}

func TestAppsMysqlEndpointValidation(t *testing.T) {
	e, _ := newTestEngine(t)
	_, r := post(t, e, "/v1/apps/mysql", `{"name":"a"}`)
	if r.Code != 5003 {
		t.Fatalf("want validation error(5003), got %+v", r)
	}
}

// 回归：DI 构造的 Redis 曾因 Ctx 为 nil 而 panic
func TestAppsRedisEndpoint(t *testing.T) {
	e, _ := newTestEngine(t)
	code, r := post(t, e, "/v1/apps/redis", ``)
	if code != http.StatusOK || r.Code != 200 || string(r.Data) != `{"acquired":true}` {
		t.Fatalf("code=%d resp=%+v data=%s", code, r, r.Data)
	}
}
