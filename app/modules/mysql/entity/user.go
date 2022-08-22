package entity

type User struct {
	ID   int64  `gorm:"column:id" db:"column:id" json:"id" form:"id"`
	Name string `gorm:"column:name" db:"column:name" json:"name" form:"name"`
	Age  int64  `gorm:"column:age" db:"column:age" json:"age" form:"age"`
}

func (u *User) TableName() string {
	return "user"
}
