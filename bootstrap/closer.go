package bootstrap

import (
	"context"
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

// closeAll 每个资源最多等待 timeout；单个失败不影响其余资源关闭
func (c *closers) closeAll(timeout time.Duration) {
	for i := len(c.items) - 1; i >= 0; i-- {
		it := c.items[i]
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := it.fn(ctx); err != nil {
			slog.Error("close failed", "component", it.name, "err", err)
		} else {
			slog.Info("closed", "component", it.name)
		}
		cancel()
	}
	c.items = nil
}
