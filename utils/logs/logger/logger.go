package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var (
	logger      *log.Logger
	fileLogger  *log.Logger
	projectRoot string
)

func init() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot = filepath.Dir(filepath.Dir(currentFile))
}

func LogToFile() error {
	var (
		logDir  = "/var/log/willshark/xlog"
		logFile = "willshark.log"
	)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		fmt.Println("日志目录不存在，使用当前目录下的 wrollup.log")
		logFile = "./" + logFile
	} else {
		fmt.Println("日志目录存在，日志将输出到:", logFile)
		logFile = filepath.Join(logDir, logFile)
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %w", err)
	}

	fileLogger = log.NewWithOptions(file, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Prefix:          "<WillShark 🚀>",
		// TODO
		Level: setLogLevel(),
	})
	fileLogger.SetFormatter(log.TextFormatter)

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.RFC3339,
		Prefix:          "<WillShark 🚀>",
	})
	logger.SetFormatter(log.TextFormatter)

	return nil
}

func setLogLevel() log.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown:0"
	}

	relPath, err := filepath.Rel(projectRoot, file)
	if err != nil {
		return fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	relPath = strings.ReplaceAll(relPath, "\\", "/")
	return fmt.Sprintf("%s:%d", relPath, line)
}

func Info(msg string) {
	caller := getCallerInfo()
	logger.Info(msg, "caller", caller)
	fileLogger.Info(msg, "caller", caller)
}

func Infof(format string, args ...interface{}) {
	/*
		format: "%s %s",
		args: []interface{}{"foo", "bar"},
	*/
	caller := getCallerInfo()
	logger.Infof(format, args, caller)
	fileLogger.Infof(format, args, caller)
}

func Error(msg string) {
	caller := getCallerInfo()
	logger.Error(msg, "caller", caller)
	fileLogger.Error(msg, "caller", caller)
}

func Debug(msg string) {
	caller := getCallerInfo()
	logger.Debug(msg, "caller", caller)
	fileLogger.Debug(msg, "caller", caller)
}

func Warn(msg string) {
	caller := getCallerInfo()
	logger.Warn(msg, "caller", caller)
	fileLogger.Warn(msg, "caller", caller)
}
