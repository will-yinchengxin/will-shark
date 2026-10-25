package dao

import (
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
	"willshark/app/do/request"
	"willshark/consts"
)

type MysqlPool struct {
	DB *gorm.DB
}

// Master 强制走主库（读写分离时用于"写后立即读"等需要强一致的查询）；未配置从库时等同于 DB
func (p *MysqlPool) Master() *gorm.DB {
	return p.DB.Clauses(dbresolver.Write)
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
