// Package request 接口请求参数（DTO），用 binding tag 校验、label tag 作为中文字段名。
package request

type Page struct {
	Page int `form:"page" json:"page" binding:"omitempty,gt=0" label:"分页"`
	Size int `form:"size" json:"size" binding:"omitempty,gt=0,lte=100" label:"每页条数"`
}
