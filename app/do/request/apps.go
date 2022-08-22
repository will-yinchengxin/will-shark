package request

type Apps struct {
	Name    string `form:"name" json:"name" validate:"required,max=20"`
	Account string `form:"account" json:"account" validate:"required,max=20"`
}
