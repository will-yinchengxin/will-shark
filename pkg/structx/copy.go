// Package structx 提供结构体字段复制工具。
package structx

import (
	"github.com/jinzhu/copier"
)

// CopyStructFields 把 b 中的同名字段复制到 a（b → a）
func CopyStructFields(a interface{}, b interface{}) error {
	return copier.Copy(a, b)
}
