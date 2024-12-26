package logs

type LogFormat interface {
	Printf(logger *Logger, logType string) error
}
