package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"willshark/pkg/errs"
	"willshark/pkg/validator"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Empty 用于无请求参数或无响应数据的接口
type Empty struct{}

// Handle 把强类型的业务函数适配为 gin.HandlerFunc：
//
//	func (c *Task) Create(ctx *gin.Context) {
//	    httpx.Handle(c.Svc.Create)(ctx)
//	}
//
// 自动完成参数绑定与校验（见 Bind）、错误映射与统一响应输出。
// 应用路由仍绑定 Controller 方法；Controller 可按需复用本适配器。
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

// Bind 依次绑定查询参数（form tag）、请求体（JSON 用 json tag，表单用 form tag）、路径参数（uri tag），
// 最后统一执行一次 binding tag 校验。文件上传等特殊场景请直接使用 gin.Context。
//
// 查询参数、表单与路径参数只绑定显式声明了对应 tag 的字段。gin 默认会把没有 tag 的字段按字段名绑定，
// 例如 `IsAdmin bool json:"-"` 可被 ?IsAdmin=true 赋值，这里过滤掉以免请求覆盖内部字段。
func Bind(c *gin.Context, obj any) error {
	// 查询参数为空时也要执行，以便应用 form tag 中的 default 值
	if err := mapTagged(obj, c.Request.URL.Query(), "form"); err != nil {
		return errs.InvalidParam.WithMsg("查询参数错误").Wrap(err)
	}
	if hasBody(c.Request) {
		if err := bindBody(c, obj); err != nil {
			return err
		}
	}
	// 路径参数最后绑定，保证 /items/:id 中的 id 不会被查询参数或请求体覆盖
	if len(c.Params) > 0 {
		m := make(map[string][]string, len(c.Params))
		for _, p := range c.Params {
			m[p.Key] = []string{p.Value}
		}
		if err := mapTagged(obj, m, "uri"); err != nil {
			return errs.InvalidParam.WithMsg("路径参数错误").Wrap(err)
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
		dec := json.NewDecoder(c.Request.Body)
		if err := dec.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return bodyError(err)
		}
		// 必须读到 EOF：拒绝第二个 JSON 值、尾部垃圾，并让尾部字节也受 BodyLimit 约束。
		if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
			if err == nil {
				return errs.InvalidParam.WithMsg("请求体只能包含一个 JSON 值")
			}
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
		if err := mapTagged(obj, c.Request.PostForm, "form"); err != nil {
			return errs.InvalidParam.WithMsg("表单参数错误").Wrap(err)
		}
	default:
		return errs.InvalidParam.WithMsg("不支持的 Content-Type: " + ct)
	}
	return nil
}

// mapTagged 逐字段绑定显式声明的 tag。仅过滤输入键不够：未打 tag 的字段可能
// 与其他字段的 tag 同名，gin 仍会按字段名将它赋值。
func mapTagged(obj any, values map[string][]string, tag string) error {
	t := reflect.TypeOf(obj)
	if t == nil {
		return fmt.Errorf("binding target is nil")
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct { // 例如绑定到 map[string]string：接收全部键
		return binding.MapFormWithTag(obj, values, tag)
	}
	_, err := mapTaggedValue(reflect.ValueOf(obj), values, tag, map[reflect.Type]bool{})
	return err
}

var taggedFieldTypes sync.Map // tagCacheKey → 单字段绑定结构体类型

type tagCacheKey struct {
	t    reflect.Type
	tag  reflect.StructTag
	name string
}

func mapTaggedValue(v reflect.Value, values map[string][]string, tag string, seen map[reflect.Type]bool) (bool, error) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if !v.CanSet() {
				return false, fmt.Errorf("binding target is nil")
			}
			next := reflect.New(v.Type().Elem())
			changed, err := mapTaggedValue(next.Elem(), values, tag, seen)
			if changed && err == nil {
				v.Set(next)
			}
			return changed, err
		}
		return mapTaggedValue(v.Elem(), values, tag, seen)
	}
	if v.Kind() != reflect.Struct || seen[v.Type()] {
		return false, nil
	}
	t := v.Type()
	seen[t] = true
	defer delete(seen, t)
	changed := false
	for i := range t.NumField() {
		f := t.Field(i)
		fv := v.Field(i)
		rawTag, explicit := f.Tag.Lookup(tag)
		name, opts, _ := strings.Cut(rawTag, ",")
		if !fv.CanSet() || name == "-" {
			continue
		}
		if explicit && name == "" {
			name = f.Name
		}
		_, present := values[name]
		if name != "" && (present || strings.Contains(opts, "default=")) {
			key := tagCacheKey{f.Type, f.Tag, f.Name}
			st, ok := taggedFieldTypes.Load(key)
			if !ok {
				st = reflect.StructOf([]reflect.StructField{{Name: f.Name, Type: f.Type, Tag: f.Tag}})
				taggedFieldTypes.Store(key, st)
			}
			one := reflect.New(st.(reflect.Type))
			one.Elem().Field(0).Set(fv)
			if err := binding.MapFormWithTag(one.Interface(), values, tag); err != nil {
				return false, err
			}
			fv.Set(one.Elem().Field(0))
			changed = true
			continue
		}
		nested, err := mapTaggedValue(fv, values, tag, seen)
		if err != nil {
			return false, err
		}
		changed = changed || nested
	}
	return changed, nil
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
