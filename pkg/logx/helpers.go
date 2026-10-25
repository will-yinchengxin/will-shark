package logx

import (
	"log/slog"
	"runtime"
	"strings"
)

func ErrorLog(err error) {
	if err == nil {
		return
	}
	slog.Error(err.Error())
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
