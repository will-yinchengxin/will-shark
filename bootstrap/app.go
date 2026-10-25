// Package bootstrap 按依赖顺序显式组装应用（替代 wire），并负责启动与优雅退出。
//
// 新增依赖时：在 New 中创建并 add 到 closers，然后注入到需要它的 service。
package bootstrap

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
	"willshark/app/daemon_jobs/cron"
	"willshark/app/modules/mysql/dao"
	rc "willshark/app/modules/redis"
	"willshark/app/router"
	"willshark/app/service"
	"willshark/consts"
	"willshark/core"
	"willshark/envconfig/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// closeTimeout 单个资源关闭的等待上限
const closeTimeout = 5 * time.Second

// Infra 基础设施连接，键为配置中的连接名
type Infra struct {
	DB    map[string]*gorm.DB
	Redis map[string]*redis.Client
	MQ    *core.RocketMQ // rocketmq.enable=false 时为 nil
}

type App struct {
	Config *config.Config
	Infra  *Infra
	Engine *gin.Engine
	Jobs   *cron.Jobs

	ready atomic.Bool
}

// New 初始化全部依赖。任一步失败会关闭已创建的资源并返回错误；
// 成功时返回的 cleanup 按初始化逆序关闭资源，应在 HTTP 服务停止后调用。
func New(ctx context.Context, cfg *config.Config) (app *App, cleanup func(), err error) {
	var cl closers
	defer func() {
		if err != nil {
			cl.closeAll(closeTimeout)
		}
	}()

	shutdownTracing, err := core.NewTracing(ctx, cfg.Tracing, cfg.App)
	if err != nil {
		return nil, nil, fmt.Errorf("init tracing: %w", err)
	}
	cl.add("tracing", shutdownTracing)

	infra := &Infra{DB: map[string]*gorm.DB{}, Redis: map[string]*redis.Client{}}
	for name, c := range cfg.MySQL {
		db, err := core.NewMySQL(ctx, c)
		if err != nil {
			return nil, nil, fmt.Errorf("init mysql %s: %w", name, err)
		}
		infra.DB[name] = db
		cl.add("mysql "+name, func(context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		})
	}
	for name, c := range cfg.Redis {
		client, err := core.NewRedis(ctx, c)
		if err != nil {
			return nil, nil, fmt.Errorf("init redis %s: %w", name, err)
		}
		infra.Redis[name] = client
		cl.add("redis "+name, func(context.Context) error { return client.Close() })
	}
	if cfg.RocketMQ.Enable {
		mq, err := core.NewRocketMQ(cfg.RocketMQ)
		if err != nil {
			return nil, nil, fmt.Errorf("init rocketmq: %w", err)
		}
		infra.MQ = mq
		cl.add("rocketmq", mq.Close)
	}

	db, ok := infra.DB[consts.DB_NAME]
	if !ok {
		return nil, nil, fmt.Errorf("mysql.%s is not configured", consts.DB_NAME)
	}
	cache, ok := infra.Redis[consts.Cache_NAME]
	if !ok {
		return nil, nil, fmt.Errorf("redis.%s is not configured", consts.Cache_NAME)
	}

	// ---- 组装业务依赖 ----
	rds := &rc.Redis{Cache: cache}
	userDao := dao.User{MysqlPool: &dao.MysqlPool{DB: db}}
	routers := &router.Routers{
		Api: &router.ApiRouter{
			Apps: &service.Apps{Rds: rds, User: userDao},
		},
	}

	app = &App{
		Config: cfg,
		Infra:  infra,
		Jobs:   &cron.Jobs{Rds: rds, User: userDao},
	}
	app.Engine, err = NewEngine(cfg, app.readiness, routers)
	if err != nil {
		return nil, nil, err
	}
	return app, func() { cl.closeAll(closeTimeout) }, nil
}

// readiness 就绪检查：退出中返回错误；依赖不可用返回错误
func (a *App) readiness(ctx context.Context) error {
	if !a.ready.Load() {
		return fmt.Errorf("not ready")
	}
	for name, db := range a.Infra.DB {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(ctx)
		}
		if err != nil {
			return fmt.Errorf("mysql %s: %w", name, err)
		}
	}
	for name, client := range a.Infra.Redis {
		if err := client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis %s: %w", name, err)
		}
	}
	return nil
}
