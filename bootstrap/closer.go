package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// closers 按注册的逆序关闭资源（后初始化的先关闭）
type closers struct {
	items []closerItem
}

type closerItem struct {
	name string
	fn   func(context.Context) error
}

func (c *closers) add(name string, fn func(context.Context) error) {
	c.items = append(c.items, closerItem{name: name, fn: fn})
}

// closeAll 为每个资源提供 timeout 期限的 context；关闭函数必须主动遵守取消。
// 单个失败（含 panic）不影响其余资源关闭，无法强制中断忽略 context 的关闭函数。
func (c *closers) closeAll(timeout time.Duration) {
	for i := len(c.items) - 1; i >= 0; i-- {
		it := c.items[i]
		if err := closeOne(it.fn, timeout); err != nil {
			slog.Error("close failed", "component", it.name, "err", err)
		} else {
			slog.Info("closed", "component", it.name)
		}
	}
	c.items = nil
}

func closeOne(fn func(context.Context) error, timeout time.Duration) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(ctx)
}
