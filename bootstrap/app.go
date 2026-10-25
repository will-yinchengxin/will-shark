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
	"willshark/app/modules/redis/lock"
	"willshark/app/router"
	"willshark/app/service"
	"willshark/consts"
	"willshark/core"
	"willshark/envconfig/config"
	"willshark/pkg/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// closeTimeout 单个资源关闭 context 的期限（关闭函数需主动遵守）
const closeTimeout = 5 * time.Second

// Infra 基础设施连接，键为配置中的连接名
type Infra struct {
	MySQL map[string]*core.MySQLConn       // 已按配置做读写分离，业务使用 .DB
	Redis map[string]redis.UniversalClient // 单机 / 集群 / 哨兵统一接口
	MQ    *core.RocketMQ                   // rocketmq.enable=false 时为 nil
}

type App struct {
	Config    *config.Config
	Infra     *Infra
	Engine    *gin.Engine
	Scheduler *scheduler.Scheduler

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

	infra := &Infra{MySQL: map[string]*core.MySQLConn{}, Redis: map[string]redis.UniversalClient{}}
	for name, c := range cfg.MySQL {
		conn, err := core.NewMySQL(ctx, c)
		if err != nil {
			return nil, nil, fmt.Errorf("init mysql %s: %w", name, err)
		}
		infra.MySQL[name] = conn
		cl.add("mysql "+name, func(context.Context) error { return conn.Close() })
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

	db, ok := infra.MySQL[consts.DBName]
	if !ok {
		return nil, nil, fmt.Errorf("mysql.%s is not configured", consts.DBName)
	}
	cache, ok := infra.Redis[consts.CacheName]
	if !ok {
		return nil, nil, fmt.Errorf("redis.%s is not configured", consts.CacheName)
	}

	app = &App{Config: cfg, Infra: infra}
	routers, sched := wire(cfg, &dao.MysqlPool{DB: db.DB}, &rc.Redis{Cache: cache})
	app.Scheduler = sched
	app.Engine, err = NewEngine(cfg, app.readiness, app.checkDeps, routers)
	if err != nil {
		return nil, nil, err
	}
	return app, func() { cl.closeAll(closeTimeout) }, nil
}

// wire 组装业务依赖：dao → service → controller → router / 定时任务。
// 注释 willshark:gen:* 为代码生成器的插入点，请保留；也可以手动在对应位置添加。
func wire(cfg *config.Config, pool *dao.MysqlPool, rds *rc.Redis) (*router.Routers, *scheduler.Scheduler) {
	// ---- dao ----
	userDao := dao.User{MysqlPool: pool}
	// willshark:gen:dao

	// ---- service ----
	appsSvc := &service.Apps{Rds: rds, User: userDao}
	// willshark:gen:service

	// ---- controller（需要直接操作 gin.Context 时使用，可选）----
	// willshark:gen:controller

	// ---- router ----
	routers := &router.Routers{
		API: &router.APIRouter{Apps: appsSvc},
		// willshark:gen:router
	}

	// ---- 定时任务：与 router 一样通过字段注入 service ----
	sched := scheduler.New(
		scheduler.WithLocker(lock.NewLocker(rds)),
		scheduler.WithLockPrefix("cron:"+cfg.App.Name+":"),
	)
	(&cron.UserStats{Apps: appsSvc}).Register(sched)
	// willshark:gen:job

	return routers, sched
}

// readiness 就绪检查（/readyz）：只反映本进程状态——启动完成且未进入退出流程。
// 不检查 MySQL / Redis 等所有副本共用的依赖：依赖抖动时如果所有 Pod 同时变为未就绪，
// 负载均衡会摘掉全部实例，局部故障被放大为整体不可用。依赖状态见 checkDeps（/healthz/deps）。
func (a *App) readiness(context.Context) error {
	if !a.ready.Load() {
		return fmt.Errorf("not ready")
	}
	return nil
}

// checkDeps 依赖健康检查（/healthz/deps）：逐个 ping 所有 MySQL 节点与 Redis，用于监控告警
func (a *App) checkDeps(ctx context.Context) error {
	for name, conn := range a.Infra.MySQL {
		if err := conn.Ping(ctx); err != nil {
			return fmt.Errorf("mysql %s: %w", name, err)
		}
	}
	for name, client := range a.Infra.Redis {
		if err := core.PingRedis(ctx, client); err != nil {
			return fmt.Errorf("redis %s: %w", name, err)
		}
	}
	return nil
}
