package validator

import (
	"github.com/go-playground/validator/v10"
	"regexp"
	"time"
)

func Phone(f validator.FieldLevel) bool {
	val := f.Field().String()
	if ok, _ := regexp.MatchString(`^1[0-9]{10}$`, val); !ok {
		return false
	}
	return true
}

func Password(f validator.FieldLevel) bool {
	val := f.Field().String()
	if ok, _ := regexp.MatchString(`[^\x00-\xff]`, val); ok {
		return false
	}
	if ok, _ := regexp.MatchString(`\s`, val); ok {
		return false
	}

	num := 0
	if ok, _ := regexp.MatchString(`[0-9]`, val); ok { //数字命中
		num++
	}
	if ok, _ := regexp.MatchString(`[a-zA-Z]`, val); ok { //字母命中
		num++
	}
	if ok, _ := regexp.MatchString(`[^A-z0-9]`, val); ok { //特殊符号命中
		num++
	}
	if num < 2 { //命中少于两次 返回错误
		return false
	}

	return true
}

// CheckUrl url checker
func CheckUrl(f validator.FieldLevel) bool {
	val := f.Field().String()
	urlPartten := "^(http|https|ftp)\\://([a-zA-Z0-9\\.\\-]+(\\:[a-zA-Z0-9\\.&amp;%\\$\\-]+)*@)*((25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9])\\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[1-9]|0)\\.(25[0-5]|2[0-4][0-9]|[0-1]{1}[0-9]{2}|[1-9]{1}[0-9]{1}|[0-9])|localhost|([a-zA-Z0-9\\-]+\\.)*[a-zA-Z0-9\\-]+\\.(com|edu|gov|int|mil|net|org|biz|arpa|info|name|pro|aero|coop|museum|[a-zA-Z]{2}))(\\:[0-9]+)*(/($|[a-zA-Z0-9\\.\\,\\?\\'\\\\\\+&amp;%\\$#\\=~_\\-]+))*$"
	if ok, _ := regexp.MatchString(urlPartten, val); ok {
		return true
	}
	return false
}

func UnixTime(f validator.FieldLevel) bool {
	t := f.Field().Int()
	format := "2006-01-02 15:04:05"
	pt, err := time.ParseInLocation(format, time.Unix(t, 0).Format(format), time.Local)
	if err != nil {
		return false
	}
	if pt.Unix() != t {
		return false
	}
	return true
}
