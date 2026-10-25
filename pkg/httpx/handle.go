package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"willshark/pkg/errs"
	"willshark/utils/validator"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Empty 用于无请求参数或无响应数据的接口
type Empty struct{}

// Handle 把强类型的业务函数适配为 gin.HandlerFunc：
//
//	g.POST("/task", httpx.Handle(svc.Task.Create))
//	func (s *Task) Create(ctx context.Context, req *request.CreateTask) (*response.Task, error)
//
// 自动完成参数绑定与校验（见 Bind）、错误映射与统一响应输出。
// ctx 为 c.Request.Context()，已包含链路追踪与 request_id。
func Handle[Req, Resp any](fn func(context.Context, *Req) (*Resp, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := new(Req)
		if err := Bind(c, req); err != nil {
			Fail(c, err)
			return
		}
		resp, err := fn(c.Request.Context(), req)
		if err != nil {
			Fail(c, err)
			return
		}
		if resp == nil {
			OK(c, nil)
			return
		}
		OK(c, resp)
	}
}

// maxMultipartMemory multipart 表单在内存中保留的上限，超出部分写临时文件
const maxMultipartMemory = 32 << 20

// Bind 依次绑定路径参数（uri tag）、查询参数（form tag）、请求体（JSON 用 json tag，表单用 form tag），
// 最后统一执行一次 binding tag 校验。文件上传等特殊场景请直接使用 gin.Context。
func Bind(c *gin.Context, obj any) error {
	if len(c.Params) > 0 {
		m := make(map[string][]string, len(c.Params))
		for _, p := range c.Params {
			m[p.Key] = []string{p.Value}
		}
		if err := binding.MapFormWithTag(obj, m, "uri"); err != nil {
			return errs.InvalidParam.WithMsg("路径参数错误").Wrap(err)
		}
	}
	// 查询参数为空时也要执行，以便应用 form tag 中的 default 值
	if err := binding.MapFormWithTag(obj, c.Request.URL.Query(), "form"); err != nil {
		return errs.InvalidParam.WithMsg("查询参数错误").Wrap(err)
	}
	if hasBody(c.Request) {
		if err := bindBody(c, obj); err != nil {
			return err
		}
	}
	if err := binding.Validator.ValidateStruct(obj); err != nil {
		if msg, ok := validator.Translate(err); ok {
			return errs.InvalidParam.WithMsg(msg).Wrap(err)
		}
		return errs.InvalidParam.Wrap(err)
	}
	return nil
}

func hasBody(r *http.Request) bool {
	return r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0
}

func bindBody(c *gin.Context, obj any) error {
	switch ct := c.ContentType(); ct {
	case binding.MIMEJSON, "":
		if err := json.NewDecoder(c.Request.Body).Decode(obj); err != nil && !errors.Is(err, io.EOF) {
			return bodyError(err)
		}
	case binding.MIMEPOSTForm, binding.MIMEMultipartPOSTForm:
		var err error
		if ct == binding.MIMEMultipartPOSTForm {
			err = c.Request.ParseMultipartForm(maxMultipartMemory)
		} else {
			err = c.Request.ParseForm()
		}
		if err != nil {
			return bodyError(err)
		}
		if err := binding.MapFormWithTag(obj, c.Request.PostForm, "form"); err != nil {
			return errs.InvalidParam.WithMsg("表单参数错误").Wrap(err)
		}
	default:
		return errs.InvalidParam.WithMsg("不支持的 Content-Type: " + ct)
	}
	return nil
}

func bodyError(err error) error {
	var (
		tooLarge *http.MaxBytesError
		typeErr  *json.UnmarshalTypeError
		syntax   *json.SyntaxError
	)
	switch {
	case errors.As(err, &tooLarge):
		return errs.TooLarge.Wrap(err)
	case errors.As(err, &typeErr):
		return errs.InvalidParam.WithMsg(fmt.Sprintf("%s 类型错误，期望类型 %s", typeErr.Field, typeErr.Type)).Wrap(err)
	case errors.As(err, &syntax), errors.Is(err, io.ErrUnexpectedEOF):
		return errs.InvalidParam.WithMsg("请求体不是合法的 JSON").Wrap(err)
	default:
		return errs.InvalidParam.WithMsg("请求体解析失败").Wrap(err)
	}
}
