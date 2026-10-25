package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"willshark/pkg/errs"
	"willshark/utils/validator"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
	if err := validator.Setup(); err != nil {
		panic(err)
	}
}

type createReq struct {
	ID    int    `uri:"id" binding:"required,gt=0"`
	Page  int    `form:"page,default=1" binding:"gte=1"`
	Name  string `json:"name" form:"name" binding:"required,max=5" label:"名称"`
	Phone string `json:"phone" form:"phone" binding:"omitempty,phoneFormat"`
}

type createResp struct {
	ID   int    `json:"id"`
	Page int    `json:"page"`
	Name string `json:"name"`
}

func newEngine(fn func(context.Context, *createReq) (*createResp, error)) *gin.Engine {
	e := gin.New()
	e.POST("/items/:id", Handle(fn))
	e.GET("/items/:id", Handle(fn))
	return e
}

func echo(_ context.Context, r *createReq) (*createResp, error) {
	return &createResp{ID: r.ID, Page: r.Page, Name: r.Name}, nil
}

func call(e *gin.Engine, method, target, contentType, body string) (int, Response) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	var r Response
	_ = json.Unmarshal(w.Body.Bytes(), &r)
	return w.Code, r
}

func TestHandleBindsURIQueryAndJSON(t *testing.T) {
	code, r := call(newEngine(echo), http.MethodPost, "/items/7?page=3", "application/json", `{"name":"abc"}`)
	if code != http.StatusOK || r.Code != CodeOK {
		t.Fatalf("code=%d resp=%+v", code, r)
	}
	data, _ := json.Marshal(r.Data)
	if string(data) != `{"id":7,"name":"abc","page":3}` {
		t.Fatalf("data=%s", data)
	}
}

func TestHandleGETUsesQueryAndDefaults(t *testing.T) {
	_, r := call(newEngine(echo), http.MethodGet, "/items/1?name=x", "", "")
	data, _ := json.Marshal(r.Data)
	if r.Code != CodeOK || string(data) != `{"id":1,"name":"x","page":1}` {
		t.Fatalf("resp=%+v data=%s", r, data)
	}
}

func TestHandleForm(t *testing.T) {
	_, r := call(newEngine(echo), http.MethodPost, "/items/1", "application/x-www-form-urlencoded", "name=f")
	if r.Code != CodeOK {
		t.Fatalf("resp=%+v", r)
	}
}

func TestHandleValidationMessagesInChinese(t *testing.T) {
	code, r := call(newEngine(echo), http.MethodPost, "/items/1", "application/json", `{"name":"toolong","phone":"123"}`)
	if code != http.StatusBadRequest || r.Code != errs.InvalidParam.Code {
		t.Fatalf("code=%d resp=%+v", code, r)
	}
	if !strings.Contains(r.Msg, "名称") || !strings.Contains(r.Msg, "1开头") {
		t.Fatalf("message should be translated with label: %q", r.Msg)
	}
}

func TestHandleBadJSON(t *testing.T) {
	e := newEngine(echo)
	if _, r := call(e, http.MethodPost, "/items/1", "application/json", `{"name":`); r.Msg != "请求体不是合法的 JSON" {
		t.Fatalf("syntax: %+v", r)
	}
	if _, r := call(e, http.MethodPost, "/items/1", "application/json", `{"name":1}`); !strings.Contains(r.Msg, "类型错误") {
		t.Fatalf("type: %+v", r)
	}
	if code, r := call(e, http.MethodPost, "/items/1", "text/xml", `<a/>`); code != http.StatusBadRequest || !strings.Contains(r.Msg, "Content-Type") {
		t.Fatalf("content-type: %d %+v", code, r)
	}
}

func TestHandleEmptyBodyPOST(t *testing.T) {
	e := gin.New()
	e.POST("/x", Handle(func(context.Context, *Empty) (*Empty, error) { return nil, nil }))
	code, r := call(e, http.MethodPost, "/x", "", "")
	data, _ := json.Marshal(r.Data)
	if code != http.StatusOK || r.Code != CodeOK || string(data) != "{}" {
		t.Fatalf("code=%d resp=%+v data=%s", code, r, data)
	}
}

func TestHandleErrorMapping(t *testing.T) {
	cases := []struct {
		err        error
		status     int
		code       int
		msg        string
		leakSubstr string
	}{
		{errs.NotFound, http.StatusNotFound, errs.NotFound.Code, errs.NotFound.Msg, ""},
		{errs.Internal.Wrap(errors.New("dsn password=secret")), http.StatusInternalServerError, errs.Internal.Code, errs.Internal.Msg, "secret"},
		{errors.New("raw sql error"), http.StatusInternalServerError, errs.Internal.Code, errs.Internal.Msg, "raw sql"},
	}
	for _, tc := range cases {
		e := newEngine(func(context.Context, *createReq) (*createResp, error) { return nil, tc.err })
		code, r := call(e, http.MethodPost, "/items/1", "application/json", `{"name":"a"}`)
		if code != tc.status || r.Code != tc.code || r.Msg != tc.msg {
			t.Fatalf("err=%v: code=%d resp=%+v", tc.err, code, r)
		}
		if tc.leakSubstr != "" && strings.Contains(r.Msg, tc.leakSubstr) {
			t.Fatalf("internal detail leaked: %q", r.Msg)
		}
	}
}

func TestAlwaysOK(t *testing.T) {
	SetAlwaysOK(true)
	defer SetAlwaysOK(false)
	e := newEngine(func(context.Context, *createReq) (*createResp, error) { return nil, errs.NotFound })
	code, r := call(e, http.MethodPost, "/items/1", "application/json", `{"name":"a"}`)
	if code != http.StatusOK || r.Code != errs.NotFound.Code {
		t.Fatalf("code=%d resp=%+v", code, r)
	}
}

func TestBodyTooLarge(t *testing.T) {
	e := gin.New()
	e.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8)
		c.Next()
	})
	e.POST("/items/:id", Handle(echo))
	code, r := call(e, http.MethodPost, "/items/1", "application/json", `{"name":"aaaaaaaaaaaaaaaa"}`)
	if code != http.StatusRequestEntityTooLarge || r.Code != errs.TooLarge.Code {
		t.Fatalf("code=%d resp=%+v", code, r)
	}
}

// 回归：没有任何查询参数时也要应用 form tag 的 default
func TestDefaultAppliedWithoutQuery(t *testing.T) {
	_, r := call(newEngine(echo), http.MethodPost, "/items/1", "application/json", `{"name":"a"}`)
	data, _ := json.Marshal(r.Data)
	if r.Code != CodeOK || !strings.Contains(string(data), `"page":1`) {
		t.Fatalf("resp=%+v data=%s", r, data)
	}
}
