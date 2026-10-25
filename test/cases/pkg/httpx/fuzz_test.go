package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 任意查询串 + JSON 请求体：绑定与校验不能 panic，只能返回 200 / 400 / 413
func FuzzHandleBind(f *testing.F) {
	f.Add("page=2&name=a", `{"name":"abc","phone":"13800138000"}`)
	f.Add("IsAdmin=1", `{"name":1}`)
	f.Add("", `[`)
	f.Fuzz(func(t *testing.T, query, body string) {
		e := newEngine(echo)
		req := httptest.NewRequest(http.MethodPost, "/items/7", strings.NewReader(body))
		if q := strings.Map(func(r rune) rune {
			if r < 0x21 || r > 0x7e || r == '#' {
				return -1
			}
			return r
		}, query); q != "" {
			req.URL.RawQuery = q
		}
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		e.ServeHTTP(w, req)
		switch w.Code {
		case http.StatusOK, http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		default:
			t.Fatalf("status %d for query=%q body=%q: %s", w.Code, query, body, w.Body.String())
		}
	})
}
