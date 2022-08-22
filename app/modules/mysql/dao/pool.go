package dao

import (
	"gorm.io/gorm"
	"will/app/do/request"
	"will/consts"
)

type MysqlPool struct {
	DB *gorm.DB
}

func Paginate(pageC request.Page) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		_, offset, pageSize := PaginateNumber(pageC)
		return db.Offset(offset).Limit(pageSize)
	}
}
func PaginateParams(offset int, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset).Limit(pageSize)
	}
}
func PaginateNumber(pageC request.Page) (page int, offset int, pageSize int) {

	page = pageC.Page
	if page == 0 {
		page = 1
	}

	pageSize = pageC.Size

	if pageC.Size <= 0 {
		pageSize = consts.PageSizeDefault
	}

	offset = (page - 1) * pageSize
	return
}
