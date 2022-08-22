package logs

import (
	"encoding/json"
	"fmt"
)

type StringFormatter struct {
	Msg     string `json:"msg"`
	AppId   string `json:"appId"`
	Env     string `json:"env"`
	LogType string `json:"logType"`
}

func (formatter StringFormatter) Printf(logger *Logger, logType string) error {
	formatter.AppId = logger.AppId
	formatter.Env = logger.Env
	formatter.LogType = logType
	mapLog := make(map[string]interface{})
	byteLog, _ := json.Marshal(formatter)
	json.Unmarshal(byteLog, &mapLog)
	fmt.Println(string(byteLog))
	return nil
}
