// Package validator 为 gin 的参数校验（binding tag）注册中文翻译与自定义规则。
//
// 请求结构体统一使用 binding tag，例如：
//
//	Phone string `json:"phone" binding:"required,phoneFormat" label:"手机号"`
//
// 错误提示中的字段名优先取 label，其次取 json 名。
package validator

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhtranslations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	setupOnce sync.Once
	setupErr  error
	trans     ut.Translator
)

type rule struct {
	tag string
	fn  validator.Func
	msg string
}

var rules = []rule{
	{"passwordFormat", Password, "{0} 密码必须为字母，数字，特殊字符中至少两种的组合"},
	{"phoneFormat", Phone, "{0} 数字组成，1开头并且长度必须为11"},
	{"urlFormat", CheckURL, "{0} 不符合url或uri的格式"},
}

// Setup 注册到 gin 默认校验器，可重复调用
func Setup() error {
	setupOnce.Do(func() { setupErr = setup() })
	return setupErr
}

func setup() error {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("gin validator engine is not go-playground/validator")
	}
	zhLocale := zh.New()
	trans, _ = ut.New(zhLocale, zhLocale).GetTranslator("zh")
	if err := zhtranslations.RegisterDefaultTranslations(v, trans); err != nil {
		return err
	}
	v.RegisterTagNameFunc(fieldLabel)

	for _, r := range rules {
		if err := v.RegisterValidation(r.tag, r.fn); err != nil {
			return err
		}
		msg, tag := r.msg, r.tag
		err := v.RegisterTranslation(tag, trans,
			func(ut ut.Translator) error { return ut.Add(tag, msg, true) },
			func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T(tag, fe.Field())
				return t
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func fieldLabel(f reflect.StructField) string {
	if label := f.Tag.Get("label"); label != "" {
		return label
	}
	if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name != "" && name != "-" {
		return name
	}
	return f.Name
}

// Translate 把校验错误翻译为中文提示；非校验错误返回 ok=false
func Translate(err error) (msg string, ok bool) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return "", false
	}
	if trans == nil {
		return ve.Error(), true
	}
	list := make([]string, 0, len(ve))
	for _, e := range ve {
		list = append(list, e.Translate(trans))
	}
	return strings.Join(list, "; "), true
}
