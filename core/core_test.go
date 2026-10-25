package core

import (
	"context"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type user struct {
	ID   int64
	Name string
}

func (user) TableName() string { return "user" }

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := RegisterTraceCallbacks(db); err != nil {
		t.Fatal(err)
	}
	return db, mock
}

// 回归：未开启链路追踪时，gorm 回调曾因 Trace 为 nil 而 panic
func TestGormCallbacksWithTracingDisabled(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	CoreConfig = map[string]interface{}{}
	initTracing()()

	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "will"))

	var u user
	if err := db.WithContext(context.Background()).First(&u, 1).Error; err != nil {
		t.Fatal(err)
	}
	if u.Name != "will" {
		t.Fatalf("got %+v", u)
	}
	// 未设置 ctx 的语句也不能 panic
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := db.Model(&user{ID: 1}).Update("name", "x").Error; err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// 开启追踪时记录 span，且 SQL 只包含占位符，不含参数值
func TestGormCallbacksRecordSpanWithoutValues(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })

	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	var u user
	_ = db.WithContext(context.Background()).Where("name = ?", "13800138000").First(&u).Error

	spans := rec.Ended()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	for _, kv := range spans[0].Attributes() {
		if kv.Key == "db.statement" {
			if strings.Contains(kv.Value.AsString(), "13800138000") {
				t.Fatalf("statement leaks parameter values: %s", kv.Value.AsString())
			}
			return
		}
	}
	t.Fatal("db.statement attribute missing")
}

func TestInitTracingWithoutConfig(t *testing.T) {
	CoreConfig = map[string]interface{}{"mysql": map[string]interface{}{}}
	stop := initTracing()
	stop()
}

func TestInitTracingEnabledRequiresEndpoint(t *testing.T) {
	CoreConfig = map[string]interface{}{"tracing": map[string]interface{}{"enable": true}}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for empty endpoint")
		}
	}()
	initTracing()
}

func TestInitTracingEnabled(t *testing.T) {
	CoreConfig = map[string]interface{}{"tracing": map[string]interface{}{
		"enable": true, "endpoint": "127.0.0.1:4318", "insecure": true,
	}}
	stop := initTracing()
	t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatal("sdk tracer provider not installed")
	}
	stop()
}

func TestGetPoolsUnknownKey(t *testing.T) {
	if _, err := GetDB("nope"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := GetRedisDB("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCorsAllowOrigins(t *testing.T) {
	CoreConfig = map[string]interface{}{"cors": map[string]interface{}{"allowOrigins": []interface{}{"https://a.com"}}}
	if got := CorsAllowOrigins(); len(got) != 1 || got[0] != "https://a.com" {
		t.Fatalf("got %v", got)
	}
	CoreConfig = map[string]interface{}{}
	if got := CorsAllowOrigins(); got != nil {
		t.Fatalf("got %v", got)
	}
}
