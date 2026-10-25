package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
	"willshark/app/middlewares"
	"willshark/app/router"
	_ "willshark/docs" // swagger 文档
	"willshark/envconfig/config"
	"willshark/pkg/httpx"
	"willshark/pkg/validator"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewEngine 创建 gin 引擎并注册中间件与路由。
// 中间件顺序：RequestID → Trace → AccessLog → Recovery → BodyLimit → CORS → 业务路由
//
// ready 用于 K8s readinessProbe（/readyz），deps 用于监控依赖状态（/healthz/deps，不要配置为探针）。
func NewEngine(cfg *config.Config, ready, deps func(context.Context) error, routers *router.Routers) (*gin.Engine, error) {
	// stdout 只输出结构化日志；gin 自身的调试输出走 stderr，且仅在本地文本日志模式下开启
	gin.DefaultWriter, gin.DefaultErrorWriter = os.Stderr, os.Stderr
	if cfg.App.Env == "dev" && cfg.Log.Format == "text" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if err := validator.Setup(); err != nil {
		return nil, err
	}
	httpx.SetAlwaysOK(cfg.HTTP.AlwaysOK)

	engine := gin.New()
	if err := engine.SetTrustedProxies(cfg.HTTP.TrustedProxies); err != nil {
		return nil, err
	}

	// 探针不经过访问日志，避免刷屏
	engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	engine.GET("/readyz", probe("readiness", ready))
	engine.GET("/healthz/deps", probe("dependency", deps))

	engine.Use(
		middlewares.RequestID(),
		middlewares.StartTrace(),
		middlewares.AccessLog(cfg.HTTP.LogRequestBody),
		middlewares.Recovery(),
		middlewares.BodyLimit(cfg.HTTP.MaxBodyBytes),
		middlewares.Cors(cfg.CORS.AllowOrigins),
	)
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, httpx.Response{Code: http.StatusNotFound, Msg: "接口不存在", Data: struct{}{}})
	})

	if cfg.App.Swagger {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	return routers.SetupRouter(engine), nil
}

// probe 检查通过返回 200，否则 503；失败原因只进日志，不返回给调用方
func probe(name string, check func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			slog.WarnContext(ctx, name+" check failed", "err", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
