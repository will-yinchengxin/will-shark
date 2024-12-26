package core

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"time"
	"willshark/consts"
	loggerv1 "willshark/utils/logs/logger"
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

var MysqlConfig map[string]interface{}

var MysqlPool map[string]*gorm.DB

func initMysql() func() {
	MysqlConfig, err := GetMapConfig(CoreConfig, "mysql", MysqlConn{})
	if err != nil {
		panic(err)
	}
	if len(MysqlConfig) < 1 {
		panic("init gorm pool config failed, mysql config not found")
	}

	MysqlPool = make(map[string]*gorm.DB)
	for name, val := range MysqlConfig {
		if db, err := initGorm(val.(*MysqlConn)); err == nil && db != nil {
			MysqlPool[name] = db
		}
	}

	return func() {
		for key, value := range MysqlPool {
			sqlDb, err := value.DB()
			err = sqlDb.Close()
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
		logrus.Fatal(err.Error())
	}
	//db = db.Debug() // start debug mod
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(conn.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conn.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Second * conn.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(time.Second * conn.ConnMaxIdletime)

	err = db.Callback().Create().Before("gorm:before_create").Register("callBackBeforeName", before)
	if err != nil {
		fmt.Println(err)
	}
	withCallback(db)
	return db, nil
}

func withCallback(db *gorm.DB) {
	_ = db.Callback().Query().Before("gorm:query").Register("callBackBeforeName", before)
	_ = db.Callback().Delete().Before("gorm:before_delete").Register("callBackBeforeName", before)
	_ = db.Callback().Update().Before("gorm:setup_reflect_value").Register("callBackBeforeName", before)
	_ = db.Callback().Row().Before("gorm:row").Register("callBackBeforeName", before)
	_ = db.Callback().Raw().Before("gorm:raw").Register("callBackBeforeName", before)

	_ = db.Callback().Create().After("gorm:after_create").Register("callBackAfterName", after)
	_ = db.Callback().Query().After("gorm:after_query").Register("callBackAfterName", after)
	_ = db.Callback().Delete().After("gorm:after_delete").Register("callBackAfterName", after)
	_ = db.Callback().Update().After("gorm:after_update").Register("callBackAfterName", after)
	_ = db.Callback().Row().After("gorm:row").Register("callBackAfterName", after)
	_ = db.Callback().Raw().After("gorm:raw").Register("callBackAfterName", after)
}

func before(db *gorm.DB) {

	_, span := Trace.Tracer("gorm").Start(db.Statement.Context, db.Statement.Table)
	db.InstanceSet(consts.JaegerStartTime, time.Now())
	db.InstanceSet(consts.JaegerGormSpanKey, span)
	return
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
	_ts, isExist := db.InstanceGet(consts.JaegerStartTime)
	if !isExist {
		return
	}
	ts, ok := _ts.(time.Time)
	if !ok {
		return
	}
	if db.Error != nil {
		span.SetAttributes(attribute.String("error", db.Error.Error()))
	}
	//sql语句
	span.SetAttributes(attribute.String("sql", db.Dialector.Explain(db.Statement.SQL.String(), db.Statement.Vars...)))
	//影响条数
	span.SetAttributes(attribute.String("affectedRows", fmt.Sprintf("%d 条", db.Statement.RowsAffected)))
	//执行时间
	span.SetAttributes(attribute.String("executeTime", fmt.Sprintf("%d ms", time.Since(ts).Milliseconds())))
	span.End()
	return
}

func GetDB(key string) (db *gorm.DB, err error) {
	db, ok := MysqlPool[key]
	if !ok {
		if config, ok := MysqlConfig[key]; !ok {
			return db, errors.New(key + " dbConfig doesn't exist")
		} else {
			if db, err = initGorm(config.(*MysqlConn)); err != nil {
				return db, errors.New(key + " dbConfig Initialization failure")
			} else {
				MysqlPool[key] = db
			}
		}
	}
	return
}
