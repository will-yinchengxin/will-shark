// Package service 实现核心业务规则、流程编排和事务，供 Controller 与定时任务复用。
package service

import (
	"context"
	"errors"
	"willshark/app/do/request"
	"willshark/app/do/response"
	"willshark/app/modules/mysql/dao"
	"willshark/app/modules/mysql/entity"
	conn "willshark/app/modules/redis"
	"willshark/app/modules/redis/lock"
	"willshark/pkg/errs"

	"gorm.io/gorm"
)

type Apps struct {
	Rds  *conn.Redis
	User dao.User
}

// List 从 MySQL 读取用户并返回响应 DTO。
func (a *Apps) List(ctx context.Context, req *request.Apps) (*response.User, error) {
	var user entity.User
	err := a.User.WithContext(ctx).Info(&user, 1)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound
	}
	if err != nil {
		return nil, errs.Internal.Wrap(err)
	}
	return &response.User{ID: user.ID, Name: user.Name, Age: user.Age}, nil
}

// Info 执行 Redis 分布式锁示例。
func (a *Apps) Info(ctx context.Context, _ *request.LockInfo) (*response.LockInfo, error) {
	clientListLock := lock.NewRedisLock(a.Rds, "client_list")
	clientListLock.SetExpire(200)
	acquired, err := clientListLock.AcquireCtx(ctx)
	if err != nil {
		return nil, errs.Internal.Wrap(err)
	}
	if !acquired {
		return &response.LockInfo{Acquired: false}, nil
	}
	released, err := clientListLock.ReleaseCtx(ctx)
	if err != nil {
		return nil, errs.Internal.Wrap(err)
	}
	return &response.LockInfo{Acquired: released}, nil
}

// CountUsers 供定时任务等内部调用
func (a *Apps) CountUsers(ctx context.Context) (int64, error) {
	n, err := a.User.WithContext(ctx).Count()
	if err != nil {
		return 0, errs.Internal.Wrap(err)
	}
	return n, nil
}
