package rand

import (
	"sync"
	"testing"
	"time"
)

func TestNextIDUniqueAndIncreasingUnderConcurrency(t *testing.T) {
	sf := NewWith(defaultEpoch, 1, 2)
	const goroutines, per = 8, 20000
	ids := make(chan int64, goroutines*per)
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			last := int64(-1)
			for range per {
				id := sf.NextID()
				if id <= last {
					t.Errorf("id not increasing: %d after %d", id, last)
					return
				}
				last = id
				ids <- id
			}
		})
	}
	wg.Wait()
	close(ids)
	seen := make(map[int64]struct{}, goroutines*per)
	for id := range ids {
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}
	}
}

func TestClockBackwardsDoesNotPanicOrRepeat(t *testing.T) {
	sf := NewWith(defaultEpoch, 0, 0)
	first := sf.NextID()
	sf.mutex.Lock()
	sf.lastTimestamp += int64(time.Hour / time.Millisecond) // 模拟时钟回拨 1 小时
	sf.mutex.Unlock()
	if second := sf.NextID(); second <= first {
		t.Fatalf("id after clock moved backwards = %d, want > %d", second, first)
	}
}

func TestNewUsesFixedEpoch(t *testing.T) {
	if a, b := New(), New(); !a.startTime.Equal(b.startTime) || !a.startTime.Equal(defaultEpoch) {
		t.Fatalf("New must use a fixed epoch: %v %v", a.startTime, b.startTime)
	}
}

func TestNewWithIDVariants(t *testing.T) {
	one := NewWith(defaultEpoch, 7)
	if one.dataCenterID != 7 || one.workerID != 7 {
		t.Fatalf("single id: %s", one)
	}
	masked := NewWith(defaultEpoch, 0xff, 0xff)
	if masked.dataCenterID != dataCenterMask || masked.workerID != workerMask {
		t.Fatalf("ids must be masked to 5 bits: %s", masked)
	}
	id := masked.NextID()
	if got := (id >> dataCenterLeftShift) & dataCenterMask; got != dataCenterMask {
		t.Fatalf("data center bits = %d", got)
	}
	_ = NewWith(defaultEpoch).String()
}
