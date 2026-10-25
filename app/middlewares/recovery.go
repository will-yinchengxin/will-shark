package middlewares

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"syscall"
	"willshark/pkg/errs"
	"willshark/pkg/httpx"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获 panic：记录带堆栈的 error 日志，返回统一的 500 响应
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if err, ok := rec.(error); ok && isBrokenPipe(err) {
				// 客户端已断开，无法再写响应
				c.Abort()
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			err := fmt.Errorf("panic: %v", rec)
			slog.ErrorContext(c.Request.Context(), "panic recovered", "err", err, "stack", string(debug.Stack()))
			httpx.Abort(c, errs.Internal.Wrap(err))
		}()
		c.Next()
	}
}

func isBrokenPipe(err error) bool {
	var se *os.SyscallError
	var ne *net.OpError
	return (errors.As(err, &ne) && errors.As(ne.Err, &se)) &&
		(errors.Is(se.Err, syscall.EPIPE) || errors.Is(se.Err, syscall.ECONNRESET))
}
