package logs

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
)

type TraceFormatter struct {
	Trace   logrus.Fields `json:"trace"`
	AppId   string        `json:"appId"`
	Env     string        `json:"env"`
	LogType string        `json:"logType"`
}

func (formatter TraceFormatter) Printf(logger *Logger, logType string) error {
	formatter.AppId = logger.AppId
	formatter.Env = logger.Env
	formatter.LogType = logType
	mapLog := make(map[string]interface{})
	byteLog, _ := json.Marshal(formatter)
	json.Unmarshal(byteLog, &mapLog)
	fmt.Println(string(byteLog))
	return nil
}
