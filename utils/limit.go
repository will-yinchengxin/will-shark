package utils

import "errors"

var ErrLimitReturn = errors.New("discarding limited token, resource pool is full, someone returned multiple times")

type Limit struct {
	pool chan struct{}
}

func NewLimit(n int) Limit {
	return Limit{
		pool: make(chan struct{}, n),
	}
}

func (l Limit) Borrow() {
	l.pool <- struct{}{}
}

func (l Limit) Return() error {
	select {
	case <-l.pool:
		return nil
	default:
		return ErrLimitReturn
	}
}

func (l Limit) TryBorrow() bool {
	select {
	case l.pool <- struct{}{}:
		return true
	default:
		return false
	}
}
