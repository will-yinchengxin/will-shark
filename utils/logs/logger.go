package logs

const (
	LOG_INFO    = "info"
	LOG_PANIC   = "panic"
	LOG_REQUEST = "request"
	LOG_ERROR   = "error"
	LOG_SUCCESS = "success"
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

func (logger *Logger) InfoDefault(info string) error {
	log := StringFormatter{
		Msg: info,
	}
	return log.Printf(logger, LOG_INFO)
}

func (logger *Logger) Error(log LogFormat) error {
	return log.Printf(logger, LOG_ERROR)
}

func (logger *Logger) ErrorDefault(info string) error {
	log := StringFormatter{
		Msg: info,
	}
	return log.Printf(logger, LOG_ERROR)
}

func (logger *Logger) Panic(log LogFormat) error {
	return log.Printf(logger, LOG_PANIC)
}

func (logger *Logger) PanicDefault(info string) error {
	log := StringFormatter{
		Msg: info,
	}
	return log.Printf(logger, LOG_PANIC)
}

func (logger *Logger) Request(log LogFormat) error {
	return log.Printf(logger, LOG_REQUEST)
}

func (logger *Logger) RequestDefault(info string) error {
	log := StringFormatter{
		Msg: info,
	}
	return log.Printf(logger, LOG_REQUEST)
}

func (logger *Logger) Success(log LogFormat) error {
	return log.Printf(logger, LOG_SUCCESS)
}

func (logger *Logger) SuccessDefault(info string) error {
	log := StringFormatter{
		Msg: info,
	}
	return log.Printf(logger, LOG_SUCCESS)
}
