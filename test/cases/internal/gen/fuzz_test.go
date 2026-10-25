package gen

import (
	"go/token"
	"os"
	"testing"
)

// 任意 SQL 文本：解析与表结构映射不能 panic；成功时字段名必须是合法的 Go 标识符
func FuzzParseSQL(f *testing.F) {
	f.Add(tricky)
	if b, err := os.ReadFile("../../sql/dynamic.sql"); err == nil {
		f.Add(string(b))
	}
	f.Add("-- c\nCREATE TABLE `t` (`id` int primary key /* x */, `n` varchar(9) COMMENT 'a''b') COMMENT='t';")
	f.Fuzz(func(t *testing.T, src string) {
		tables, err := ParseSQL(src)
		if err != nil {
			return
		}
		for _, tb := range tables {
			td, err := newTableData(tb)
			if td == nil {
				continue
			}
			_ = err
			for _, fl := range td.Fields {
				if !token.IsIdentifier(fl.Name) {
					t.Fatalf("field name %q from column %q is not a Go identifier", fl.Name, fl.Column)
				}
			}
		}
	})
}

// 任意名称：NewName 成功时各种写法都必须是合法的 Go 标识符或路径片段
func FuzzNewName(f *testing.F) {
	for _, s := range []string{"order_item", "HTTPServer", "tenantId", "a-b", "x9", "type"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := NewName(s)
		if err != nil {
			return
		}
		if !token.IsIdentifier(n.Pascal) || !token.IsIdentifier(n.Camel) || token.IsKeyword(n.Camel) || !token.IsExported(n.Pascal) {
			t.Fatalf("NewName(%q) = %+v", s, n)
		}
	})
}
