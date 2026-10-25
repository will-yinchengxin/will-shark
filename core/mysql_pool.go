package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"willshark/consts"
	loggerv1 "willshark/utils/logs/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type MysqlConn struct {
	Host            string        `yaml:"host,omitempty"`
	Port            int           `yaml:"port,omitempty"`
	User            string        `yaml:"user,omitempty"`
	PassWord        string        `yaml:"password,omitempty"`
	DataBase        string        `yaml:"dataBase,omitempty"`
	MaxIdleConns    int           `yaml:"maxIdleConns,omitempty"`
	MaxOpenConns    int           `yaml:"maxOpenConns,omitempty"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime,omitempty"`
	ConnMaxIdletime time.Duration `yaml:"connMaxIdletime,omitempty"`
}

// MysqlPool 启动时一次性初始化，之后只读
var MysqlPool map[string]*gorm.DB

func initMysql() func() {
	mysqlConfig, err := GetMapConfig(CoreConfig, "mysql", MysqlConn{})
	if err != nil {
		panic(err)
	}
	if len(mysqlConfig) < 1 {
		panic("init gorm pool config failed, mysql config not found")
	}

	MysqlPool = make(map[string]*gorm.DB)
	for name, val := range mysqlConfig {
		db, err := initGorm(val.(*MysqlConn))
		if err != nil {
			panic("init mysql [" + name + "] failed: " + err.Error())
		}
		MysqlPool[name] = db
	}

	return func() {
		for key, value := range MysqlPool {
			sqlDb, err := value.DB()
			if err == nil {
				err = sqlDb.Close()
			}
			if err != nil {
				loggerv1.Error("MySQL[ " + key + " ]Closed Err:" + err.Error())
				continue
			}
			loggerv1.Info("MySQL[ " + key + " ]Closed Success!")
		}
	}
}

func initGorm(conn *MysqlConn) (*gorm.DB, error) {
	connectStr := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		conn.User, conn.PassWord, conn.Host, conn.Port, conn.DataBase)

	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       connectStr,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, //关闭表名复数
		},
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold: time.Second * 3, // 慢 SQL 阈值
				LogLevel:      logger.Silent,   // Log level
				Colorful:      false,           // 禁用彩色打印
			},
		),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(conn.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conn.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Second * conn.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(time.Second * conn.ConnMaxIdletime)

	if err = RegisterTraceCallbacks(db); err != nil {
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

func GetDB(key string) (*gorm.DB, error) {
	db, ok := MysqlPool[key]
	if !ok {
		return nil, errors.New("mysql [" + key + "] not initialized, check the mysql config")
	}
	return db, nil
}
