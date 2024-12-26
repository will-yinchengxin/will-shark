package dao

import (
	"context"
	"willshark/app/modules/mysql/entity"
)

type User struct {
	*MysqlPool
}

func (u *User) WithContext(ctx context.Context) *User {
	u.DB = u.DB.WithContext(ctx)
	return u
}

func (u *User) Info(userInfo *entity.User, uid int) error {
	if err := u.DB.Model(entity.User{}).
		Where("id = ?", uid).
		First(&userInfo).Error; err != nil {
		return err
	}
	return nil
}
