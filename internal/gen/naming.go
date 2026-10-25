package gen

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
)

// Name 一个标识符的各种写法
type Name struct {
	Raw    string
	Pascal string // OrderItem / UserID
	Camel  string // orderItem / userID
	Snake  string // order_item
	Kebab  string // order-item
}

// initialisms Go 惯用的全大写缩写
var initialisms = map[string]bool{
	"ACL": true, "API": true, "ASCII": true, "CPU": true, "CSS": true, "DNS": true, "EOF": true,
	"GUID": true, "HTML": true, "HTTP": true, "HTTPS": true, "ID": true, "IP": true, "JSON": true,
	"QPS": true, "RAM": true, "RPC": true, "SDK": true, "SLA": true, "SMTP": true, "SQL": true,
	"SSH": true, "TCP": true, "TLS": true, "TTL": true, "UDP": true, "UI": true, "UID": true,
	"URI": true, "URL": true, "UTF8": true, "UUID": true, "VM": true, "XML": true,
}

// NewName 解析 order_item / order-item / orderItem / OrderItem 等写法
func NewName(s string) (Name, error) {
	words := splitWords(s)
	if len(words) == 0 {
		return Name{}, fmt.Errorf("invalid name %q", s)
	}
	for _, w := range words {
		for _, r := range w {
			if r >= unicode.MaxASCII {
				return Name{}, fmt.Errorf("invalid name %q: only ASCII letters, digits, '_' and '-' are supported", s)
			}
		}
	}
	if !unicode.IsLetter(rune(words[0][0])) {
		return Name{}, fmt.Errorf("invalid name %q: must start with a letter", s)
	}
	n := Name{Raw: s}
	var pascal, camel strings.Builder
	lower := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
		p := pascalWord(w)
		pascal.WriteString(p)
		if i == 0 {
			camel.WriteString(lower[i])
		} else {
			camel.WriteString(p)
		}
	}
	n.Pascal, n.Camel = pascal.String(), camel.String()
	n.Snake = strings.Join(lower, "_")
	n.Kebab = strings.Join(lower, "-")
	if token.IsKeyword(n.Camel) {
		n.Camel += "_"
	}
	return n, nil
}

func pascalWord(w string) string {
	up := strings.ToUpper(w)
	if initialisms[up] {
		return up
	}
	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

// splitWords 按分隔符与大小写边界拆分：HTTPServer → HTTP Server，tenantId → tenant Id
func splitWords(s string) []string {
	var words []string
	var cur []rune
	rs := []rune(s)
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	for i, r := range rs {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
		case unicode.IsUpper(r):
			prevLower := i > 0 && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]))
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			prevUpper := i > 0 && unicode.IsUpper(rs[i-1])
			if prevLower || (prevUpper && nextLower) {
				flush()
			}
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
	}
	flush()
	return words
}

// fieldName 列名转 Go 字段名；数字开头的列名加前缀 F，非 ASCII 列名返回错误
func fieldName(col string) (string, error) {
	n, err := NewName(col)
	if err != nil {
		if n, err = NewName("f_" + col); err != nil {
			return "", fmt.Errorf("column %q: %w", col, err)
		}
	}
	if n.Pascal == "TableName" {
		return "TableNameField", nil
	}
	return n.Pascal, nil
}
