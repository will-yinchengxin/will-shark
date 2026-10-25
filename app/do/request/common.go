package request

type Page struct {
	Page int `form:"page" json:"page" binding:"omitempty,gt=0" label:"分页"`
	Size int `form:"size" json:"size" binding:"omitempty,gt=0" label:"每页条数"`
}
