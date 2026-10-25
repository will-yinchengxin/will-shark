package middlewares

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

// 任意请求体：脱敏不能 panic；输出要么是合法 JSON，要么是占位符，且敏感字段的值不会出现在输出中
func FuzzRedactJSON(f *testing.F) {
	f.Add([]byte(`{"password":"p1","user":{"token":"t1","list":[{"phone":"138"}]}}`))
	f.Add([]byte(`[1,"a",null]`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		out := RedactJSON(raw)
		if out == "<non-json body>" {
			return
		}
		var v any
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatalf("output is not JSON: %q", out)
		}
		var walk func(any)
		walk = func(x any) {
			switch x := x.(type) {
			case map[string]any:
				for k, val := range x {
					if _, ok := sensitiveKeys[strings.ToLower(k)]; ok && val != "***" {
						t.Fatalf("sensitive key %q not redacted: %q", k, out)
					}
					walk(val)
				}
			case []any:
				for _, e := range x {
					walk(e)
				}
			}
		}
		walk(v)
	})
}

// 任意请求 ID：只接受长度 ≤64 的可见 ASCII（防日志注入）
func FuzzValidRequestID(f *testing.F) {
	f.Add("abc-123")
	f.Add("a\nb")
	f.Fuzz(func(t *testing.T, id string) {
		if !validRequestID(id) {
			return
		}
		if len(id) == 0 || len(id) > 64 || !utf8.ValidString(id) {
			t.Fatalf("accepted %q", id)
		}
		for _, r := range id {
			if r < 0x21 || r > 0x7e {
				t.Fatalf("accepted control/non-ASCII char in %q", id)
			}
		}
	})
}
