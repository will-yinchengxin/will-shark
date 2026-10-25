package validator

import (
	"net/url"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var (
	phoneRe       = regexp.MustCompile(`^1[0-9]{10}$`)
	nonASCIIRe    = regexp.MustCompile(`[^\x00-\xff]`)
	spaceRe       = regexp.MustCompile(`\s`)
	digitRe       = regexp.MustCompile(`[0-9]`)
	letterRe      = regexp.MustCompile(`[a-zA-Z]`)
	specialCharRe = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

func Phone(f validator.FieldLevel) bool {
	return phoneRe.MatchString(f.Field().String())
}

// Password 不含中文与空白，且数字、字母、特殊字符至少包含两种；长度请配合 min/max 规则
func Password(f validator.FieldLevel) bool {
	val := f.Field().String()
	if nonASCIIRe.MatchString(val) || spaceRe.MatchString(val) {
		return false
	}
	num := 0
	for _, re := range []*regexp.Regexp{digitRe, letterRe, specialCharRe} {
		if re.MatchString(val) {
			num++
		}
	}
	return num >= 2
}

// CheckURL 校验 http / https / ftp 绝对地址（必须带主机名）
func CheckURL(f validator.FieldLevel) bool {
	u, err := url.Parse(f.Field().String())
	if err != nil || u.Hostname() == "" {
		return false
	}
	switch u.Scheme {
	case "http", "https", "ftp":
		return true
	}
	return false
}
