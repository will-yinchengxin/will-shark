package cron

import (
	"context"
	"testing"
	"time"
	"willshark/app/modules/mysql/dao"
	"willshark/app/service"
	"willshark/pkg/scheduler"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 任务通过注入的 service 访问数据库，可以像普通对象一样单测
func TestUserStatsUsesService(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user`").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	job := &UserStats{Apps: &service.Apps{User: dao.User{MysqlPool: &dao.MysqlPool{DB: db}}}}
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type noopLocker struct{}

func (noopLocker) TryLock(context.Context, string, time.Duration) (func(context.Context), bool, error) {
	return func(context.Context) {}, true, nil
}

func TestUserStatsRegister(t *testing.T) {
	s := scheduler.New(scheduler.WithLocker(noopLocker{}))
	(&UserStats{}).Register(s)
	if names := s.Names(); len(names) != 1 || names[0] != "user_stats" {
		t.Fatalf("names = %v", names)
	}
}

func TestUserStatsReturnsServiceError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db, _ := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{})
	mock.ExpectQuery("SELECT count").WillReturnError(context.DeadlineExceeded)
	job := &UserStats{Apps: &service.Apps{User: dao.User{MysqlPool: &dao.MysqlPool{DB: db}}}}
	if err := job.Run(context.Background()); err == nil {
		t.Fatal("db error must be returned so the scheduler logs it")
	}
}
