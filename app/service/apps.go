package service

import (
	"context"
	"will/app/do/request"
	"will/app/modules/mysql/dao"
	"will/app/modules/mysql/entity"
	conn "will/app/modules/redis"
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
	name := a.Rds.Get("name1")
	return name, &utils.CodeType{}
}
