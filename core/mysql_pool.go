package core

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"time"
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
				_ = Log.PanicDefault("MySQL[ " + key + " ]Closed Err:" + err.Error())
				continue
			}
			_ = Log.SuccessDefault("MySQL[ " + key + " ]Closed Success!")
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

	// Todo: do something about db trace

	return db, nil
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
