package utils

import (
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"runtime"
	"strconv"
	"strings"
	"will/consts"
	"will/core"
	"will/will_tools/logs"
)

func ErrorLog(err error) {
	if err == nil {
		return
	}
	// 0 represents the caller of the caller (the call stack where the caller is located)
	// 1 represents the caller who calls the caller and so on
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	f := runtime.FuncForPC(pc)
	_ = core.Log.Error(logs.TraceFormatter{
		Trace: logrus.Fields{
			"err":      err.Error(),
			"file":     file,
			"line":     strconv.Itoa(line),
			"function": f.Name(),
		},
	})

	return
}

func GetMqRealTag(tag string) (real string) {
	if core.RocketConfig == nil {
		return tag
	}
	topic := core.RocketConfig.Topic
	if r, ok := topic[tag]; ok {
		return r
	}
	return tag
}

// copy a by b  b->a
func CopyStructFields(a interface{}, b interface{}, fields ...string) (err error) {
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

func InsertSliceHead() {
	a := []int{1, 2, 3, 4, 5}
	a = append(a, 0)
	index := 2
	copy(a[index+1:], a[index:])
	a[index] = 0
}
