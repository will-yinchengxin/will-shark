package service

import (
	"context"
	"fmt"
	"willshark/app/do/request"
	"willshark/app/modules/mysql/dao"
	"willshark/app/modules/mysql/entity"
	conn "willshark/app/modules/redis"
	"willshark/app/modules/redis/lock"
	"willshark/utils"
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
	return nil, &utils.CodeType{}
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
