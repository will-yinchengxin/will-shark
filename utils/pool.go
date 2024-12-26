package utils

import (
	"github.com/panjf2000/ants/v2"
	"sync"
)

type Pool struct {
	pool *ants.Pool
	wg   sync.WaitGroup
}

func NewPool(size int) (*Pool, error) {
	p, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &Pool{
		pool: p,
	}, nil
}

func (p *Pool) Submit(task func()) error {
	p.wg.Add(1)
	return p.pool.Submit(func() {
		defer p.wg.Done()
		task()
	})
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) Running() int {
	return p.pool.Running()
}

func (p *Pool) Cap() int {
	return p.pool.Cap()
}

func (p *Pool) Free() int {
	return p.pool.Free()
}

func (p *Pool) Release() {
	p.pool.Release()
}

func (p *Pool) Reboot() {
	p.pool.Reboot()
}

/*
pool, err := wgpool.NewPool(10)
    if err != nil {
        panic(err)
    }
    defer pool.Release()

    // 提交任务
    for i := 0; i < 20; i++ {
        i := i
        err := pool.Submit(func() {
            fmt.Printf("Task %d is running\n", i)
            time.Sleep(100 * time.Millisecond)
        })
        if err != nil {
            fmt.Printf("Submit task %d failed: %v\n", i, err)
        }
    }

    // 等待所有任务完成
    pool.Wait()
    fmt.Println("All tasks completed")
*/
