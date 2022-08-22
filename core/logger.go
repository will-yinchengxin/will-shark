package core

import (
	"strconv"
	"will/consts"
	"will/will_tools/logs"
)

var Log *logs.Logger

func initLogger() func() {
	Log = logs.NewLogger(strconv.Itoa(consts.APP_ID), Environment)
	return func() {}
}
