// Package syncx 提供 goroutine 池、限流信号量、条件通知与原子值。
// 新代码中的原子布尔值可直接使用标准库 sync/atomic.Bool。
package syncx

import "sync/atomic"

// Boolean is a boolean value, all actions of it is atomic
type Boolean uint32

// Get reads the value atomically
func (b *Boolean) Get() bool {
	return atomic.LoadUint32((*uint32)(b)) != 0
}

// Set writes the value atomically
func (b *Boolean) Set(v bool) {
	if v {
		atomic.StoreUint32((*uint32)(b), 1)
	} else {
		atomic.StoreUint32((*uint32)(b), 0)
	}
}
