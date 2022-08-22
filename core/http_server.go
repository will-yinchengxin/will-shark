package core

import (
	"context"
	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
	"will/consts"
	"will/will_tools/logs"
)

func HttpStarter(engine *gin.Engine) func() {
	if Environment == "dev" {
		server, _ := initHTTPServer(context.TODO(), engine)
		logInfo := logs.TraceFormatter{
			Trace: logrus.Fields{
				"welcome will's gang": "start the service with http in dev environment",
			},
		}
		_ = Log.Info(logInfo)
		err := server.ListenAndServe()
		if err != nil {
			logrus.Fatal(err.Error())
		}
	} else {
		logInfo := logs.TraceFormatter{
			Trace: logrus.Fields{
				"welcome will's gang": "start the service with endless server in other environment",
			},
		}
		_ = Log.Info(logInfo)
		err := endless.ListenAndServe(":"+consts.SERVER_PORT, engine)
		if err != nil {
			logrus.Fatal(err.Error())
		}
	}

	return func() {}
}

// you can set your own http.Handler in dev environment
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
			logInfo := logs.StringFormatter{Msg: err.Error()}
			Log.Error(logInfo)
		}
	}
}
