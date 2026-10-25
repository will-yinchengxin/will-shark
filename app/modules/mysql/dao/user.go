package dao

import (
	"context"
	"willshark/app/modules/mysql/entity"
)

type User struct {
	*MysqlPool
}

// WithContext 返回绑定了 ctx 的新实例，不修改共享的 DB，避免并发请求间串用 ctx
func (u *User) WithContext(ctx context.Context) *User {
	return &User{MysqlPool: &MysqlPool{DB: u.DB.WithContext(ctx)}}
}

func (u *User) Info(userInfo *entity.User, uid int) error {
	return u.DB.Model(&entity.User{}).
		Where("id = ?", uid).
		First(userInfo).Error
}
