package service

import (
	"context"
	"errors"
	"willshark/app/do/request"
	"willshark/app/modules/mysql/dao"
	"willshark/app/modules/mysql/entity"
	conn "willshark/app/modules/redis"
	"willshark/app/modules/redis/lock"
	"willshark/utils"

	"gorm.io/gorm"
)

type Apps struct {
	Rds  *conn.Redis
	User dao.User
}

func (a *Apps) List(req request.Apps, ctx context.Context) (res interface{}, codeType *utils.CodeType) {
	var (
		err   error
		param entity.User
	)
	defer func() {
		utils.ErrorLog(err)
	}()
	err = a.User.WithContext(ctx).Info(&param, 1)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = nil
		return nil, utils.NotFoundError
	}
	if err != nil {
		return nil, utils.DBError
	}
	return param, &utils.CodeType{}
}

func (a *Apps) Info(ctx context.Context) (res interface{}, codeType *utils.CodeType) {
	var (
		err error
	)
	defer func() {
		utils.ErrorLog(err)
	}()
	clientListLock := lock.NewRedisLock(a.Rds, "client_list")
	clientListLock.SetExpire(200)
	acquired, err := clientListLock.AcquireCtx(ctx)
	if err != nil {
		return nil, utils.FAILURE
	}
	if !acquired {
		return utils.SetRetVal("acquired", false), &utils.CodeType{}
	}
	released, err := clientListLock.ReleaseCtx(ctx)
	if err != nil {
		return nil, utils.FAILURE
	}
	return utils.SetRetVal("acquired", released), &utils.CodeType{}
}
