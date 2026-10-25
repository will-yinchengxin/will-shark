package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Run 监听配置端口并阻塞，直到 ctx 取消（收到退出信号）或服务出错
func (a *App) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(a.Config.HTTP.Port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return a.Serve(ctx, ln)
}

// Serve 在给定 listener 上提供服务。退出流程：
//  1. /readyz 返回 503，等待 http.drainWait，让负载均衡摘除流量
//  2. 停止接收新连接，等待在途请求完成（最长 http.shutdownTimeout）
//  3. 停止定时任务并等待当前一轮执行结束
//
// 数据库等资源由 New 返回的 cleanup 在 Serve 返回后关闭。
func (a *App) Serve(ctx context.Context, ln net.Listener) error {
	hc := a.Config.HTTP
	srv := &http.Server{
		Handler:           a.Engine,
		ReadHeaderTimeout: hc.ReadHeaderTimeout,
		ReadTimeout:       hc.ReadTimeout,
		WriteTimeout:      hc.WriteTimeout,
		IdleTimeout:       hc.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}

	jobsCtx, stopJobs := context.WithCancel(context.WithoutCancel(ctx))
	defer stopJobs()
	a.Jobs.Start(jobsCtx)

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	a.ready.Store(true)
	slog.Info("http server started", "addr", ln.Addr().String(), "env", a.Config.App.Env)

	var runErr error
	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-serveErr:
		runErr = fmt.Errorf("http server: %w", err)
	}
	a.ready.Store(false)

	if runErr == nil {
		if hc.DrainWait > 0 {
			slog.Info("draining before shutdown", "wait", hc.DrainWait.String())
			time.Sleep(hc.DrainWait)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), hc.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			runErr = fmt.Errorf("http shutdown: %w", err)
		} else {
			slog.Info("http server stopped")
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) && runErr == nil {
			runErr = err
		}
	}

	stopJobs()
	jobsWaitCtx, cancel := context.WithTimeout(context.Background(), hc.ShutdownTimeout)
	defer cancel()
	if err := a.Jobs.Wait(jobsWaitCtx); err != nil {
		slog.Warn("cron jobs did not stop in time", "err", err)
	}
	return runErr
}
