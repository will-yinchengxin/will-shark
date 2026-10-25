package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestJSONRequiresSingleValue(t *testing.T) {
	for _, body := range []string{`{"name":"a"} {"name":"b"}`, `{"name":"a"} garbage`} {
		code, _ := call(newEngine(echo), http.MethodPost, "/items/1", "application/json", body)
		if code != http.StatusBadRequest {
			t.Errorf("body=%q status=%d, want 400", body, code)
		}
	}
}

func TestExplicitEmptyTagNameKeepsDefaults(t *testing.T) {
	type req struct {
		Age      int `form:",default=18"`
		Internal int
	}
	for _, tc := range []struct {
		query string
		age   int
	}{{"", 18}, {"?Age=21&Internal=7", 21}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/"+tc.query, nil)
		var got req
		if err := Bind(c, &got); err != nil || got.Age != tc.age || got.Internal != 0 {
			t.Fatalf("%s: got=%+v err=%v", tc.query, got, err)
		}
	}
}

func TestJSONTrailingBytesRespectBodyLimit(t *testing.T) {
	e := gin.New()
	e.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32)
	})
	e.POST("/items/:id", Handle(echo))
	code, _ := call(e, http.MethodPost, "/items/1", "application/json", `{"name":"a"}`+strings.Repeat(" ", 100))
	if code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d, want 413", code)
	}
}

func TestTaggedKeyCannotBindUntaggedSibling(t *testing.T) {
	type req struct {
		Name   string
		Public string `form:"Name"`
		Hidden struct {
			Admin bool `form:"admin"`
		} `form:"-"`
	}
	e := gin.New()
	var got req
	e.POST("/", Handle(func(_ context.Context, r *req) (*Empty, error) { got = *r; return nil, nil }))
	code, _ := call(e, http.MethodPost, "/?Name=visible&admin=true", "", "")
	if code != http.StatusOK || got.Public != "visible" || got.Name != "" || got.Hidden.Admin {
		t.Fatalf("status=%d got=%+v", code, got)
	}
}
