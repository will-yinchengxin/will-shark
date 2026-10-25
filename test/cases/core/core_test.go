package core

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
	"willshark/envconfig/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/apache/rocketmq-client-go/v2"
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

func TestNewTracingDisabledIsNoop(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	shutdown, err := NewTracing(context.Background(), config.Tracing{}, config.App{Name: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		t.Fatal("disabled tracing must not install sdk provider")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewTracingEnabled(t *testing.T) {
	t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })
	shutdown, err := NewTracing(context.Background(),
		config.Tracing{Enable: true, Endpoint: "127.0.0.1:4318", Insecure: true, SampleRatio: 1},
		config.App{Name: "t", ID: "1", Env: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Fatal("sdk tracer provider not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = shutdown(ctx) // 无 collector 时导出失败可忽略，只要求不阻塞
}

func TestNewRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	client, err := NewRedis(context.Background(), config.Redis{Addr: mr.Addr()})
	if err != nil {
		t.Fatal(err)
	}
	_ = client.Close()

	addr := mr.Addr()
	mr.Close()
	if _, err := NewRedis(context.Background(), config.Redis{Addr: addr, DialTimeout: 200 * time.Millisecond}); err == nil {
		t.Fatal("expected error for unreachable redis")
	}
}

func TestNewMySQLUnreachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	start := time.Now()
	_, err = NewMySQL(context.Background(), config.MySQL{Host: "127.0.0.1", Port: addr.Port, User: "u", Database: "d"})
	if err == nil {
		t.Fatal("expected error for unreachable mysql")
	}
	if time.Since(start) > 10*time.Second {
		t.Fatal("connect should fail fast")
	}
}

// 未启动消费者时 Close 不能 panic（rocketmq PushConsumer 未 Start 就 Shutdown 会空指针）
func TestRocketMQCloseWithoutStartingConsumer(t *testing.T) {
	mq, err := NewRocketMQ(config.RocketMQ{Enable: true, NameServers: []string{"127.0.0.1:1"}, GroupName: "test", Retry: 1})
	if err != nil {
		t.Skipf("rocketmq client: %v", err)
	}
	if err := mq.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type fakeConsumer struct {
	rocketmq.PushConsumer
	started, shutdown int
}

func (f *fakeConsumer) Start() error    { f.started++; return nil }
func (f *fakeConsumer) Shutdown() error { f.shutdown++; return nil }

type fakeProducer struct {
	rocketmq.Producer
	shutdown int
}

func (f *fakeProducer) Shutdown() error { f.shutdown++; return nil }

func TestRocketMQStartConsumerThenClose(t *testing.T) {
	c, p := &fakeConsumer{}, &fakeProducer{}
	mq := &RocketMQ{Producer: p, Consumer: c}
	if err := mq.Close(context.Background()); err != nil || c.shutdown != 0 || p.shutdown != 1 {
		t.Fatalf("unstarted consumer must not be shut down: %+v %+v", c, p)
	}
	if err := mq.StartConsumer(); err != nil || c.started != 1 {
		t.Fatal(err)
	}
	if err := mq.Close(context.Background()); err != nil || c.shutdown != 1 {
		t.Fatalf("started consumer must be shut down: %+v", c)
	}
}

func TestNewRocketMQInvalidNameServer(t *testing.T) {
	if _, err := NewRocketMQ(config.RocketMQ{Enable: true, NameServers: []string{"not a valid addr"}, GroupName: "g"}); err == nil {
		t.Fatal("invalid name server must fail")
	}
}
