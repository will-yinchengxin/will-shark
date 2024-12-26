package core

import (
	"context"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"willshark/consts"
	"willshark/utils/logs/logger"
)

const (
	startStr = `
 __        ___ _ _     ____  _                _    
 \ \      / (_) | |   / ___|| |__   __ _ _ __| | __
  \ \ /\ / /| | | |   \___ \| '_ \ / _  | '__| |/ /
   \ V  V / | | | |    ___) | | | | (_| | |  |   <
    \_/\_/  |_|_|_|   |____/|_| |_|\__,_|_|  |_|\_\
`
)

// HttpStarter
func HttpStarter(engine *gin.Engine) (*http.Server, func()) {
	server, stopServer := initHTTPServer(context.Background(), engine)
	color.Green(startStr)
	color.Green("Server Port: " + consts.SERVER_PORT)
	return server, stopServer
}

// initHTTPServer
// You can set your own http.Handler in dev environment
func initHTTPServer(ctx context.Context, handler http.Handler) (*http.Server, func()) {
	srv := &http.Server{
		Addr:         ":" + consts.SERVER_PORT,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return srv, func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(30))
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error(err.Error())
		}
	}
}
