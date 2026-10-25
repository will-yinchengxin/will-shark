package utils

import (
	"strings"
	"testing"
	"time"
)

func TestPoolWaitDoesNotHangAfterFailedSubmit(t *testing.T) {
	p, err := NewPool(1)
	if err != nil {
		t.Fatal(err)
	}
	p.Release()
	if err := p.Submit(func() {}); err == nil {
		t.Fatal("submit to a released pool should fail")
	}
	done := make(chan struct{})
	go func() { p.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait hangs after a failed Submit")
	}
}

func TestFillMissingTimePointsRejectsBadInput(t *testing.T) {
	type point struct{ TimePoint uint32 }
	if _, err := FillMissingTimePoints([]point{{1}}, 0, 10, -1); err == nil {
		t.Fatal("negative interval must be rejected (was an infinite loop)")
	}
	if _, err := FillMissingTimePoints([]int{1}, 0, 10, 1); err == nil {
		t.Fatal("non-struct elements must be rejected (was a panic)")
	}
	out, err := FillMissingTimePoints([]point{{2}}, 0, 6, 2)
	if err != nil || len(out.([]point)) != 3 {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestGetTimeSeriesShortRow(t *testing.T) {
	if _, err := GetTimeSeries([][]string{{"2024-03-02 00:00:00"}}); err == nil {
		t.Fatal("row with a single element must return an error (was an index panic)")
	}
}

func TestGetTimeStrSlice(t *testing.T) {
	cases := map[[2]string]string{
		{"2026-01-01", "2026-01-01"}: "2026-01-01",
		{"2026-01-30", "2026-02-02"}: "2026-01-30,2026-01-31,2026-02-01,2026-02-02",
	}
	for in, want := range cases {
		if got := strings.Join(GetTimeStrSlice(in[0], in[1]), ","); got != want {
			t.Errorf("GetTimeStrSlice(%s, %s) = %s, want %s", in[0], in[1], got, want)
		}
	}
}

func TestCondSignal(t *testing.T) {
	c := NewCond()
	go func() {
		time.Sleep(10 * time.Millisecond)
		c.Signal()
	}()
	if _, ok := c.WaitWithTimeout(time.Second); !ok {
		t.Fatal("Signal must wake the waiter")
	}
	c.Signal() // 无等待者时不阻塞
}

func TestTimeHelpersRejectBadInput(t *testing.T) {
	if got := ParseTimeStrToInt("bad"); got != 0 {
		t.Errorf("ParseTimeStrToInt(bad) = %d, want 0", got)
	}
	if got := GetTimeStrSlice("x", "2026-01-01"); len(got) != 0 {
		t.Errorf("GetTimeStrSlice(bad) = %v, want empty", got)
	}
	for _, in := range []string{"1.5d", "-3d", "d", "10x"} {
		if _, err := AgoUnixMilli(in); err == nil {
			t.Errorf("AgoUnixMilli(%q) should fail", in)
		}
	}
	got, err := AgoUnixMilli("2d")
	if want := time.Now().AddDate(0, 0, -2).UnixMilli(); err != nil || got > want || want-got > 1000 {
		t.Errorf("AgoUnixMilli(2d) = %d, %v", got, err)
	}
}

func TestFillMissingTimePointsRejectsOutOfRange(t *testing.T) {
	type point struct{ TimePoint uint32 }
	if _, err := FillMissingTimePoints([]point{{1}}, -10, 10, 1); err == nil {
		t.Fatal("negative start must be rejected (would be truncated into uint32)")
	}
	if _, err := FillMissingTimePoints([]point{{1}}, 0, 1<<33, 1<<30); err == nil {
		t.Fatal("end beyond uint32 must be rejected")
	}
}

func TestRandHelpers(t *testing.T) {
	seen := map[int64]bool{}
	for range 100 {
		seen[GetRandInt64()] = true
	}
	if len(seen) < 99 {
		t.Fatalf("random values repeat: %d distinct of 100", len(seen))
	}
	if a, b := NewRand().Int63(), NewRand().Int63(); a == b {
		t.Fatal("independent generators must be seeded differently")
	}
}
