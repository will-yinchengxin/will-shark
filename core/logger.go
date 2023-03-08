package core

import (
	"will/consts"
	"will/will_tools/logs"
)

var Log *logs.Logger

func initLogger() func() {
	Log = logs.NewLogger(consts.APP_ID, Environment)
	return func() {}
}
