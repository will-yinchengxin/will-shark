package logs

const (
	LOG_INFO    = "info"
	LOG_PANIC   = "panic"
	LOG_REQUEST = "request"
	LOG_ERROR   = "error"
)

type Logger struct {
	AppId string
	Env   string
}

func NewLogger(appId string, env string) *Logger {
	return &Logger{AppId: appId, Env: env}
}

func (logger *Logger) Info(log LogFormat) error {
	return log.Printf(logger, LOG_INFO)
}

func (logger *Logger) Error(log LogFormat) error {
	return log.Printf(logger, LOG_ERROR)
}

func (logger *Logger) Panic(log LogFormat) error {
	return log.Printf(logger, LOG_PANIC)
}

func (logger *Logger) Request(log LogFormat) error {
	return log.Printf(logger, LOG_REQUEST)
}
