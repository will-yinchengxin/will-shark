package logs

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
)

type HttpFormatter struct {
	Header  logrus.Fields `json:"header"`
	Body    logrus.Fields `json:"body"`
	Trace   logrus.Fields `json:"trace"`
	AppId   string        `json:"appId"`
	Env     string        `json:"env"`
	LogType string        `json:"logType"`
}

func (formatter HttpFormatter) Printf(logger *Logger, logType string) error {
	formatter.AppId = logger.AppId
	formatter.Env = logger.Env
	formatter.LogType = logType
	mapLog := make(map[string]interface{})
	byteLog, _ := json.Marshal(formatter)
	json.Unmarshal(byteLog, &mapLog)
	fmt.Println(string(byteLog))
	return nil
}
