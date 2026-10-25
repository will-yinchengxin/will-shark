package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // 内置时区数据，容器内无需安装 tzdata
	"willshark/bootstrap"
	"willshark/envconfig/config"
	"willshark/pkg/logx"
)

// version 构建时通过 -ldflags "-X main.version=..." 注入
var version = "dev"

const banner = `
 __        ___ _ _     ____  _                _
 \ \      / (_) | |   / ___|| |__   __ _ _ __| | __
  \ \ /\ / /| | | |   \___ \| '_ \ / _  | '__| |/ /
   \ V  V / | | | |    ___) | | | | (_| | |  |   <
    \_/\_/  |_|_|_|   |____/|_| |_|\__,_|_|  |_|\_\
`

// 用法：willshark [env]，env 也可由环境变量 APP_ENV 指定，默认 dev
func main() {
	if err := run(); err != nil {
		slog.Error("exit with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	env := config.ResolveEnv(os.Args[1:])
	cfg, err := config.Load(env)
	if err != nil {
		return err
	}
	logx.Setup(os.Stdout, cfg.Log.Level, cfg.Log.Format,
		"service", cfg.App.Name, "env", env, "version", version)
	if cfg.Log.Format == "text" {
		fmt.Fprint(os.Stderr, banner)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop() // 恢复默认信号处理：再次 Ctrl-C 立即退出
	}()

	app, cleanup, err := bootstrap.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup()
	return app.Run(ctx)
}
