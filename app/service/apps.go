package service

import (
	"context"
	"fmt"
	"will/app/do/request"
	"will/app/modules/mysql/dao"
	"will/app/modules/mysql/entity"
	conn "will/app/modules/redis"
	"will/app/modules/redis/lock"
	"will/utils"
)

type Apps struct {
	Rds  *conn.RedisPool
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

	userModel := a.User.WithContext(ctx)
	err = userModel.Info(&param, 1)
	if err != nil {
		return nil, utils.NotFoundError
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
	clientListAcquire, err := clientListLock.Acquire()

	if err != nil || !clientListAcquire {
		fmt.Println(err, clientListAcquire, "clientListAcquire")
		return nil, &utils.CodeType{}
	}
	release, err := clientListLock.Release()
	if !release || err != nil {
		return nil, &utils.CodeType{}
	}
	return nil, &utils.CodeType{}
}
