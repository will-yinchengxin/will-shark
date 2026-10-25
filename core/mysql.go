package core

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"time"
	"willshark/consts"
	"willshark/envconfig/config"

	mysqldriver "github.com/go-sql-driver/mysql"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// NewMySQL 创建 gorm 连接池并验证连通性
func NewMySQL(ctx context.Context, conf config.MySQL) (*gorm.DB, error) {
	dsn := mysqldriver.NewConfig()
	dsn.User = conf.User
	dsn.Passwd = conf.Password
	dsn.Net = "tcp"
	dsn.Addr = net.JoinHostPort(conf.Host, strconv.Itoa(conf.Port))
	dsn.DBName = conf.Database
	dsn.ParseTime = true
	dsn.Loc = time.Local
	dsn.Collation = "utf8mb4_unicode_ci" // 连接字符集 utf8mb4，支持 emoji 与生僻字
	dsn.Timeout = 5 * time.Second

	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn.FormatDSN()}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, //关闭表名复数
		},
		// 慢 SQL 与错误进 slog（自动带 trace_id）；只打印占位符 SQL，不打印参数值
		Logger: logger.NewSlogLogger(slog.Default(), logger.Config{
			SlowThreshold:             conf.SlowThreshold,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(conf.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(conf.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := RegisterTraceCallbacks(db); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// RegisterTraceCallbacks 为 gorm 注册链路追踪回调
func RegisterTraceCallbacks(db *gorm.DB) error {
	cb := db.Callback()
	return errors.Join(
		cb.Create().Before("gorm:before_create").Register("callBackBeforeName", before),
		cb.Query().Before("gorm:query").Register("callBackBeforeName", before),
		cb.Delete().Before("gorm:before_delete").Register("callBackBeforeName", before),
		cb.Update().Before("gorm:setup_reflect_value").Register("callBackBeforeName", before),
		cb.Row().Before("gorm:row").Register("callBackBeforeName", before),
		cb.Raw().Before("gorm:raw").Register("callBackBeforeName", before),

		cb.Create().After("gorm:after_create").Register("callBackAfterName", after),
		cb.Query().After("gorm:after_query").Register("callBackAfterName", after),
		cb.Delete().After("gorm:after_delete").Register("callBackAfterName", after),
		cb.Update().After("gorm:after_update").Register("callBackAfterName", after),
		cb.Row().After("gorm:row").Register("callBackAfterName", after),
		cb.Raw().After("gorm:raw").Register("callBackAfterName", after),
	)
}

// before 使用全局 tracer，未开启链路追踪时为 noop，不会 panic
func before(db *gorm.DB) {
	ctx := db.Statement.Context
	if ctx == nil {
		ctx = context.Background()
	}
	_, span := otel.Tracer("gorm").Start(ctx, db.Statement.Table)
	db.InstanceSet(consts.JaegerStartTime, time.Now())
	db.InstanceSet(consts.JaegerGormSpanKey, span)
}

func after(db *gorm.DB) {
	_span, isExist := db.InstanceGet(consts.JaegerGormSpanKey)
	if !isExist {
		return
	}
	span, ok := _span.(trace.Span)
	if !ok {
		return
	}
	defer span.End()
	if !span.IsRecording() {
		return
	}
	if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
		span.RecordError(db.Error)
		span.SetStatus(codes.Error, db.Error.Error())
	}
	// 只记录带占位符的 SQL，不记录参数值，避免敏感数据进入链路系统
	span.SetAttributes(
		attribute.String("db.statement", db.Statement.SQL.String()),
		attribute.Int64("db.rows_affected", db.Statement.RowsAffected),
	)
	if ts, ok := db.InstanceGet(consts.JaegerStartTime); ok {
		if start, ok := ts.(time.Time); ok {
			span.SetAttributes(attribute.Int64("db.execute_ms", time.Since(start).Milliseconds()))
		}
	}
}
