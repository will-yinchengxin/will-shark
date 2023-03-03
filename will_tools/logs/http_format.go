package logs

import (
	"encoding/json"
	"github.com/fatih/color"
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
	byteLog, _ := json.Marshal(formatter)

	switch logType {
	case LOG_INFO:
		color.White(string(byteLog))
	case LOG_PANIC, LOG_ERROR:
		color.Red(string(byteLog))
	case LOG_REQUEST:
		color.Blue(string(byteLog))
	case LOG_SUCCESS:
		color.Green(string(byteLog))
	}
	return nil
}
