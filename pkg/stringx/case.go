// Package stringx 提供字符串转换与编码工具。
package stringx

import (
	"unicode"
	"unicode/utf8"
)

// LowerFirst 首字符转小写（按 rune 处理，支持多字节字符）
func LowerFirst(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if size == 0 {
		return ""
	}
	return string(unicode.ToLower(r)) + s[size:]
}
