package validator

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"strings"
)

type ValidatorX struct {
	Validate *validator.Validate
	uni      *ut.UniversalTranslator
	trans    ut.Translator
}

func NewValidator() *ValidatorX {
	validate := validator.New()
	zh := zh.New()
	uni := ut.New(zh, zh)
	trans, _ := uni.GetTranslator("zh")
	_ = zh_translations.RegisterDefaultTranslations(validate, trans)
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		label := field.Tag.Get("label")
		if label == "" {
			return field.Name
		}
		return label
	})
	v := &ValidatorX{
		Validate: validate,
		uni:      uni,
		trans:    trans,
	}
	v.Register()
	return v
}

func (v *ValidatorX) Register() {
	_ = v.Validate.RegisterValidation("passwordFormat", Password)
	_ = v.Validate.RegisterValidation("phoneFormat", Phone)
	_ = v.Validate.RegisterValidation("urlFormat", CheckUrl)

	_ = v.Validate.RegisterTranslation("passwordFormat", v.trans, func(ut ut.Translator) error {
		return ut.Add("passwordFormat", "{0} 密码必须为字母，数字，特殊字符中至少两种的组合", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("passwordFormat", fe.Field())
		return t
	})

	_ = v.Validate.RegisterTranslation("phoneFormat", v.trans, func(ut ut.Translator) error {
		return ut.Add("phoneFormat", "{0} 数字组成，1开头并且长度必须为11", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("phoneFormat", fe.Field())
		return t
	})

	_ = v.Validate.RegisterTranslation("urlFormat", v.trans, func(ut ut.Translator) error {
		return ut.Add("urlFormat", "{0} 不符合url或uri的格式", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("urlFormat", fe.Field())
		return t
	})
}

func (v *ValidatorX) Translate(errs validator.ValidationErrors) string {
	var errList []string
	for _, e := range errs {
		// can translate each error one at a time.
		errList = append(errList, e.Translate(v.trans))
	}
	return strings.Join(errList, "|")
}

// ParseQuery
func (v *ValidatorX) ParseQuery(c *gin.Context, obj interface{}) string {
	if err := c.ShouldBindQuery(obj); err != nil {
		return "参数解析失败"
	}
	err := v.Validate.Struct(obj)
	if err != nil {
		return v.parseErrorHandler(err)
	}

	return ""
}

// parse form data
func (v *ValidatorX) ParseForm(c *gin.Context, obj interface{}) string {
	if err := c.ShouldBindWith(obj, binding.Form); err != nil {
		return "参数解析失败"
	}
	err := v.Validate.Struct(obj)
	if err != nil {
		return v.parseErrorHandler(err)
	}

	return ""
}

// parse json data
func (v *ValidatorX) ParseJson(c *gin.Context, obj interface{}) string {
	if err := c.ShouldBindWith(obj, binding.JSON); err != nil {
		return "参数解析失败," + err.Error()
	}
	err := v.Validate.Struct(obj)
	if err != nil {
		return v.parseErrorHandler(err)
	}
	return ""
}

// parse form data, use gin shouldBind method
func (v *ValidatorX) ParseHeader(c *gin.Context, obj interface{}) string {
	if err := c.ShouldBindHeader(obj); err != nil {
		return "参数解析失败"
	}
	err := v.Validate.Struct(obj)
	if err != nil {
		return v.parseErrorHandler(err)
	}

	return ""
}

func (v *ValidatorX) parseErrorHandler(err error) string {
	var errStr string
	switch err.(type) {
	case validator.ValidationErrors:
		errStr = v.Translate(err.(validator.ValidationErrors))
	case *json.UnmarshalTypeError:
		unmarshalTypeError := err.(*json.UnmarshalTypeError)
		errStr = fmt.Errorf("%s 类型错误，期望类型 %s", unmarshalTypeError.Field, unmarshalTypeError.Type.String()).Error()
	default:
		errStr = err.Error()
	}
	return errStr
}
