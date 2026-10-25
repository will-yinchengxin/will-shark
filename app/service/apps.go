// Package service 业务逻辑层：强类型方法供 router（httpx.Handle）、controller 与定时任务复用。
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

// List 示例：从 MySQL 读取用户
//
//	@Summary	获取App信息
//	@Description	通过 mysql 获取信息
//	@Tags		App相关
//	@Accept		json
//	@Produce	json
//	@Param		body	body		request.Apps	true	"请求参数"
//	@Success	200		{object}	httpx.Response{data=entity.User}
//	@Router		/v1/apps/mysql [post]
func (a *Apps) List(ctx context.Context, req *request.Apps) (*entity.User, error) {
	var user entity.User
	err := a.User.WithContext(ctx).Info(&user, 1)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound
	}
	if err != nil {
		return nil, errs.Internal.Wrap(err)
	}
	return &user, nil
}

// Info 示例：Redis 分布式锁
//
//	@Summary	获取App信息
//	@Description	通过 redis 获取信息
//	@Tags		App相关
//	@Produce	json
//	@Success	200	{object}	httpx.Response{data=response.LockInfo}
//	@Router		/v1/apps/redis [post]
func (a *Apps) Info(ctx context.Context, _ *struct{}) (*response.LockInfo, error) {
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
