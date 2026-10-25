package errs

import (
	"runtime"
	"strconv"

	"github.com/pkg/errors"
)

func WithMessage(err error, message string) error {
	return errors.WithMessage(err, "==> "+printCallerNameAndLine()+message)
}

func printCallerNameAndLine() string {
	pc, _, line, _ := runtime.Caller(2)
	return runtime.FuncForPC(pc).Name() + "()@line:" + strconv.Itoa(line) + ": "
}
