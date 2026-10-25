package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	"gorm.io/plugin/dbresolver"
)

// MySQLConn 一个逻辑数据库：DB 供业务使用（已按配置做读写分离），nodes 用于健康检查与关闭
type MySQLConn struct {
	DB    *gorm.DB
	nodes []mysqlNode
}

type mysqlNode struct {
	role string // primary | source | replica
	addr string
	db   *sql.DB
}

// Ping 检查所有节点（主库、额外主库、从库）
func (c *MySQLConn) Ping(ctx context.Context) error {
	for _, n := range c.nodes {
		if err := n.db.PingContext(ctx); err != nil {
			return fmt.Errorf("%s %s: %w", n.role, n.addr, err)
		}
	}
	return nil
}

// Close 关闭所有节点的连接池
func (c *MySQLConn) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	for _, n := range c.nodes {
		if err := n.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", n.role, n.addr, err))
		}
	}
	return errors.Join(errs...)
}

// NewMySQL 为主库及 sources / replicas 中的每个节点建立连接池并验证连通性。
// 配置了 sources 或 replicas 时启用 dbresolver：
//   - SELECT（非事务）→ replicas（未配置 replicas 时走主库）
//   - 写操作、事务、db.Clauses(dbresolver.Write) → 主库与 sources
func NewMySQL(ctx context.Context, conf config.MySQL) (_ *MySQLConn, err error) {
	conn := &MySQLConn{}
	defer func() {
		if err != nil {
			_ = conn.Close() // 关闭已建立的节点
		}
	}()
	open := func(role string, n config.MySQLNode) (*sql.DB, error) {
		db, err := openNode(ctx, conf, n)
		if err != nil {
			return nil, fmt.Errorf("%s %s:%d: %w", role, n.Host, n.Port, err)
		}
		conn.nodes = append(conn.nodes, mysqlNode{role: role, addr: net.JoinHostPort(n.Host, strconv.Itoa(n.Port)), db: db})
		return db, nil
	}

	primary, err := open("primary", conf.Node())
	if err != nil {
		return nil, err
	}
	var sources, replicas []*sql.DB
	for _, n := range conf.Sources {
		db, err := open("source", n)
		if err != nil {
			return nil, err
		}
		sources = append(sources, db)
	}
	for _, n := range conf.Replicas {
		db, err := open("replica", n)
		if err != nil {
			return nil, err
		}
		replicas = append(replicas, db)
	}

	conn.DB, err = newGorm(conf, primary, sources, replicas)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func openNode(ctx context.Context, conf config.MySQL, n config.MySQLNode) (*sql.DB, error) {
	dsn := mysqldriver.NewConfig()
	dsn.User = n.User
	dsn.Passwd = n.Password
	dsn.Net = "tcp"
	dsn.Addr = net.JoinHostPort(n.Host, strconv.Itoa(n.Port))
	dsn.DBName = n.Database
	dsn.ParseTime = true
	dsn.Loc = time.Local
	dsn.Collation = "utf8mb4_unicode_ci" // 连接字符集 utf8mb4，支持 emoji 与生僻字
	dsn.Timeout = 5 * time.Second

	connector, err := mysqldriver.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(conf.MaxIdleConns)
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetConnMaxLifetime(conf.ConnMaxLifetime)
	db.SetConnMaxIdleTime(conf.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// newGorm 基于已建立的连接池创建 gorm 实例并注册读写分离与链路追踪
func newGorm(conf config.MySQL, primary *sql.DB, sources, replicas []*sql.DB) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: primary, SkipInitializeWithVersion: true}), &gorm.Config{
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

	if len(sources) > 0 || len(replicas) > 0 {
		resolver := dbresolver.Config{Policy: dbresolver.RandomPolicy{}}
		if conf.Policy == "round_robin" {
			resolver.Policy = dbresolver.RoundRobinPolicy()
		}
		// 显式配置 Sources 时 dbresolver 不再使用主连接写入，因此需把主库也加入
		if len(sources) > 0 {
			resolver.Sources = append(resolver.Sources, dialector(primary))
			for _, s := range sources {
				resolver.Sources = append(resolver.Sources, dialector(s))
			}
		}
		for _, r := range replicas {
			resolver.Replicas = append(resolver.Replicas, dialector(r))
		}
		if err := db.Use(dbresolver.Register(resolver)); err != nil {
			return nil, fmt.Errorf("register dbresolver: %w", err)
		}
	}

	if err := RegisterTraceCallbacks(db); err != nil {
		return nil, err
	}
	return db, nil
}

func dialector(conn *sql.DB) gorm.Dialector {
	return mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true})
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

// WrapGorm 包装已有的 gorm 实例为 MySQLConn（测试或自定义连接时使用）
func WrapGorm(db *gorm.DB) (*MySQLConn, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return &MySQLConn{DB: db, nodes: []mysqlNode{{role: "primary", addr: "custom", db: sqlDB}}}, nil
}
