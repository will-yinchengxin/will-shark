// Package utils 通用工具函数。
package utils

import (
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"willshark/consts"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func ErrorLog(err error) {
	if err == nil {
		return
	}
	slog.Error(err.Error())
}

// CopyStructFields 把 b 中的同名字段复制到 a（b → a）
func CopyStructFields(a interface{}, b interface{}) error {
	return copier.Copy(a, b)
}

func WithMessage(err error, message string) error {
	return errors.WithMessage(err, "==> "+printCallerNameAndLine()+message)
}

func printCallerNameAndLine() string {
	pc, _, line, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name() + "()@line:" + strconv.Itoa(line) + ": "
}

func GetSelfFuncName() string {
	pc, _, _, _ := runtime.Caller(1)
	return cleanUpFuncName(runtime.FuncForPC(pc).Name())
}

func cleanUpFuncName(funcName string) string {
	end := strings.LastIndex(funcName, ".")
	if end == -1 {
		return ""
	}
	return funcName[end+1:]
}

func SetPassword(password string) ([]byte, error) {
	newPass, err := bcrypt.GenerateFromPassword([]byte(password), consts.PasswordDifficult)
	return newPass, err
}
