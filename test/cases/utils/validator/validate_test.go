package validator

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestCustomRules(t *testing.T) {
	v := validator.New()
	for _, r := range rules {
		if err := v.RegisterValidation(r.tag, r.fn); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		tag, val string
		ok       bool
	}{
		{"phoneFormat", "13800138000", true},
		{"phoneFormat", "1380013800", false},
		{"passwordFormat", "abc12345", true},
		{"passwordFormat", "abc_defg", true}, // 下划线是特殊字符（旧正则 [^A-z0-9] 不认）
		{"passwordFormat", "abcdefgh", false},
		{"passwordFormat", "abc 1234", false},
		{"passwordFormat", "密码abc123", false},
		{"urlFormat", "https://api.example.dev:8443/v1/x?a=1&b=2", true},
		{"urlFormat", "http://10.0.0.1/path", true},
		{"urlFormat", "https://example.xyz", true},
		{"urlFormat", "javascript:alert(1)", false},
		{"urlFormat", "/relative/path", false},
		{"urlFormat", "https://", false},
	}
	for _, c := range cases {
		if err := v.Var(c.val, c.tag); (err == nil) != c.ok {
			t.Errorf("%s(%q) ok=%v, want %v", c.tag, c.val, err == nil, c.ok)
		}
	}
}
