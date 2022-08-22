package utils

import (
	"github.com/sirupsen/logrus"
	"runtime"
	"strconv"
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
