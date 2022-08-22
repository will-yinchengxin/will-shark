package request

type Page struct {
	Page int `form:"page" json:"page" validate:"omitempty,gt=0" label:"分页"`
	Size int `form:"size" json:"size" validate:"omitempty,gt=0" label:"每页条数"`
}
