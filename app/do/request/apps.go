package request

type Apps struct {
	Name    string `form:"name" json:"name" binding:"required,max=20"`
	Account string `form:"account" json:"account" binding:"required,max=20"`
}
