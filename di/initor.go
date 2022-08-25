package di

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/now"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"io"
	"io/ioutil"
	"strconv"
	"will/app/daemon_jobs/cron"
	"will/app/middlewares"
	"will/app/modules/mysql/dao"
	"will/app/modules/redis"
	"will/app/router"
	"will/consts"
	"will/core"
	"will/will_tools/logs"
)

func InitGinEngine(router *router.Routers) (*gin.Engine, func()) {
	gin.DefaultWriter = io.MultiWriter()
	engine := gin.Default()

	engine.Use(func(c *gin.Context) {
		bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
		c.Request.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		var bodyMap map[string]interface{}
		_ = json.Unmarshal(bodyBytes, &bodyMap)
		c.Set("bodyMap", bodyMap)
	})
	engine.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		header := make(map[string]interface{})
		for k, v := range param.Request.Header {
			header[k] = v[0]
		}

		body := make(map[string]interface{})
		if len(param.Keys["bodyMap"].(map[string]interface{})) > 0 {
			body = param.Keys["bodyMap"].(map[string]interface{})
		} else {
			for k, v := range param.Request.Form {
				body[k] = v[0]
			}
		}
		logMap := logrus.Fields{
			"clientIP":    param.ClientIP,
			"timestamp":   param.TimeStamp.Unix(),
			"time":        now.New(param.TimeStamp).Format("2006-01-02 15:04:05"),
			"method":      param.Method,
			"path":        param.Path,
			"proto":       param.Request.Proto,
			"status":      param.StatusCode,
			"duration":    strconv.Itoa(int(param.Latency.Milliseconds())) + "ms",
			"userAgent":   param.Request.UserAgent(),
			"errorMsg":    param.ErrorMessage,
			"resBodySize": param.BodySize,
			"resData":     param.Keys["resData"],
		}
		trace := logMap
		logInfo := logs.HttpFormatter{
			Header: header,
			Body:   body,
			Trace:  trace,
		}
		_ = core.Log.Request(logInfo)
		return ""
	}))
	// set swagger doc tools
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// set the global middleware
	engine.Use(middlewares.Cors())
	// init router
	engine = router.SetupRouter(engine)

	return engine, func() {}
}

func InitMysql() (*dao.MysqlPool, func()) {
	dbPool, err := core.GetDB(consts.DB_NAME_WILL)
	if err != nil {
		panic("db is error")
		return nil, func() {}
	}
	return &dao.MysqlPool{dbPool}, func() {}
}

func InitRedis() (*redis.RedisPool, func()) {
	redisPool, err := core.GetRedisDB("will")
	if err != nil {
		return nil, func() {}
	}

	return &redis.RedisPool{redisPool, redisPool.Get()}, func() {}
}

func InitCronJobs() *cron.Jobs {
	ctx, cancel := context.WithCancel(context.Background())
	return &cron.Jobs{
		Ctx:       ctx,
		CancelFun: cancel,
	}
}
