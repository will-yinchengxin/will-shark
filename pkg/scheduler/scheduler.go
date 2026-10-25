// Package scheduler 进程内定时任务调度。
//
// 任务就是普通的函数 func(ctx) error。业务上推荐把任务写成一个持有依赖（service / dao）的结构体，
// 和 router 一样在 bootstrap 中组装并注册：
//
//	type OrderCleanup struct{ Order *service.Order }
//
//	func (j *OrderCleanup) Register(s *scheduler.Scheduler) {
//		s.Every("order_cleanup", 5*time.Minute, j.Run, scheduler.WithLock(time.Minute))
//	}
//
//	func (j *OrderCleanup) Run(ctx context.Context) error { return j.Order.CleanExpired(ctx) }
//
// 特性：
//   - Every 固定间隔（上一轮结束后开始计时）/ Cron 表达式（支持可选的秒字段与 @every、@daily 等）
//   - 同一任务在单进程内串行执行，不会重叠
//   - WithLock：多副本部署时借助分布式锁保证同一时刻只有一个实例执行
//   - WithTimeout：单次执行超时；WithRunOnStart：启动后立即执行一次
//   - Trigger：手动触发（例如管理接口），通过 channel 通知任务循环，不阻塞调用方
//   - panic 隔离、执行耗时与错误日志、每次执行一个 trace span
//   - ctx 取消后不再开始新一轮，Wait 等待正在执行的任务结束
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// Func 任务函数；返回的 error 会被记录日志
type Func func(ctx context.Context) error

// Locker 分布式锁。ok=false 表示锁被其他实例持有，本轮跳过
type Locker interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (unlock func(context.Context), ok bool, err error)
}

// ErrUnknownJob Trigger 了未注册的任务
var ErrUnknownJob = errors.New("scheduler: unknown job")

type Scheduler struct {
	locker     Locker
	lockPrefix string

	mu      sync.Mutex
	jobs    map[string]*job
	started atomic.Bool
	wg      sync.WaitGroup
}

type job struct {
	name       string
	desc       string
	schedule   cron.Schedule
	fn         Func
	timeout    time.Duration
	lockTTL    time.Duration
	runOnStart bool
	trigger    chan struct{}
}

// Option 任务选项
type Option func(*job)

// WithTimeout 单次执行超时，超时后 ctx 被取消（任务需响应 ctx）
func WithTimeout(d time.Duration) Option { return func(j *job) { j.timeout = d } }

// WithLock 执行前获取分布式锁，ttl 应大于任务最长执行时间；需在 New 时提供 WithLocker
func WithLock(ttl time.Duration) Option { return func(j *job) { j.lockTTL = ttl } }

// WithRunOnStart 调度器启动后立即执行一次
func WithRunOnStart() Option { return func(j *job) { j.runOnStart = true } }

// SchedulerOption 调度器选项
type SchedulerOption func(*Scheduler)

// WithLocker 设置分布式锁实现
func WithLocker(l Locker) SchedulerOption { return func(s *Scheduler) { s.locker = l } }

// WithLockPrefix 锁 key 前缀，默认 "scheduler:"；建议包含应用名以免与其他服务冲突
func WithLockPrefix(p string) SchedulerOption { return func(s *Scheduler) { s.lockPrefix = p } }

func New(opts ...SchedulerOption) *Scheduler {
	s := &Scheduler{jobs: map[string]*job{}, lockPrefix: "scheduler:"}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Every 注册固定间隔任务：上一轮结束后等待 interval 再执行下一轮。
// 注册阶段的编程错误（重名、启动后注册、缺少 Locker）会 panic，以便在启动时暴露。
func (s *Scheduler) Every(name string, interval time.Duration, fn Func, opts ...Option) {
	if interval <= 0 {
		panic(fmt.Sprintf("scheduler: job %q interval must be positive", name))
	}
	s.add(name, "every "+interval.String(), cron.ConstantDelaySchedule{Delay: interval}, fn, opts)
}

// cronParser 支持 5 段（分 时 日 月 周）、6 段（秒 分 时 日 月 周）与 @every/@daily 等描述符
var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// Cron 注册 cron 表达式任务，时区为进程本地时区（容器内由 TZ 决定）
func (s *Scheduler) Cron(name, spec string, fn Func, opts ...Option) {
	sched, err := cronParser.Parse(spec)
	if err != nil {
		panic(fmt.Sprintf("scheduler: job %q invalid cron spec %q: %v", name, spec, err))
	}
	s.add(name, "cron "+spec, sched, fn, opts)
}

func (s *Scheduler) add(name, desc string, sched cron.Schedule, fn Func, opts []Option) {
	if s.started.Load() {
		panic(fmt.Sprintf("scheduler: job %q registered after Start", name))
	}
	j := &job{name: name, desc: desc, schedule: sched, fn: fn, trigger: make(chan struct{}, 1)}
	for _, o := range opts {
		o(j)
	}
	if j.lockTTL > 0 && s.locker == nil {
		panic(fmt.Sprintf("scheduler: job %q uses WithLock but no Locker configured", name))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.jobs[name]; dup {
		panic(fmt.Sprintf("scheduler: duplicate job %q", name))
	}
	s.jobs[name] = j
}

// Names 已注册的任务名（排序后）
func (s *Scheduler) Names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.jobs))
	for n := range s.jobs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Start 为每个任务启动一个调度循环；ctx 取消后不再开始新一轮
func (s *Scheduler) Start(ctx context.Context) {
	if !s.started.CompareAndSwap(false, true) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		s.wg.Go(func() { s.loop(ctx, j) })
		slog.Info("cron job scheduled", "job", j.name, "schedule", j.desc)
	}
}

// Trigger 请求立即执行一次任务（异步）。任务正在执行或已有待执行的触发时合并为一次。
func (s *Scheduler) Trigger(name string) error {
	s.mu.Lock()
	j, ok := s.jobs[name]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownJob, name)
	}
	select {
	case j.trigger <- struct{}{}:
	default: // 已有待处理的触发
	}
	return nil
}

// Wait 等待所有任务循环退出；超过 ctx 期限返回错误
func (s *Scheduler) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait cron jobs: %w", ctx.Err())
	}
}

func (s *Scheduler) loop(ctx context.Context, j *job) {
	if j.runOnStart && ctx.Err() == nil {
		s.run(ctx, j)
	}
	for {
		timer := time.NewTimer(time.Until(j.schedule.Next(time.Now())))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		case <-j.trigger:
			timer.Stop()
		}
		if ctx.Err() != nil {
			return
		}
		s.run(ctx, j)
	}
}

func (s *Scheduler) run(ctx context.Context, j *job) {
	ctx, span := otel.Tracer("scheduler").Start(ctx, "cron "+j.name)
	defer span.End()
	log := slog.With("job", j.name)

	if j.lockTTL > 0 {
		unlock, ok, err := s.locker.TryLock(ctx, s.lockPrefix+j.name, j.lockTTL)
		if err != nil {
			log.WarnContext(ctx, "cron job lock failed, skipped", "err", err)
			return
		}
		if !ok {
			log.DebugContext(ctx, "cron job skipped, locked by another instance")
			return
		}
		defer unlock(context.WithoutCancel(ctx))
	}

	runCtx := ctx
	if j.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, j.timeout)
		defer cancel()
	}

	start := time.Now()
	err := safeCall(runCtx, j.fn)
	elapsed := time.Since(start)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.ErrorContext(ctx, "cron job failed", "err", err, "elapsed", elapsed.String())
		return
	}
	log.InfoContext(ctx, "cron job done", "elapsed", elapsed.String())
}

func safeCall(ctx context.Context, fn Func) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	return fn(ctx)
}
