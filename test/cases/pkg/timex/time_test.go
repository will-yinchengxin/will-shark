package timex

import (
	"testing"
	"time"
)

func TestGetTimeSeries(t *testing.T) {
	ok, err := GetTimeSeries([][]string{
		{"2024-03-03 00:00:00", "2024-03-03 01:00:00"},
		{"2024-03-02 00:00:00", "2024-03-02 01:00:00"},
	})
	if err != nil || len(ok) != 2 || !ok[0].StartTime.Before(ok[1].StartTime) {
		t.Fatalf("sorted series: %v %v", ok, err)
	}
	for name, in := range map[string][][]string{
		"overlap":     {{"2024-03-02 00:00:00", "2024-03-02 02:00:00"}, {"2024-03-02 01:00:00", "2024-03-02 03:00:00"}},
		"too early":   {{"2023-01-01 00:00:00", "2023-01-01 01:00:00"}},
		"bad format":  {{"2024/03/02", "2024-03-02 01:00:00"}},
		"bad end fmt": {{"2024-03-02 00:00:00", "x"}},
	} {
		if _, err := GetTimeSeries(in); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestTimeFormat(t *testing.T) {
	ts := ParseTimeStrToInt("2024-03-02 10:20:30")
	if ParseTimeIntToStr(ts) != "2024-03-02 10:20:30" || ParseTimeIntToStrOnlyDay(ts) != "2024-03-02" {
		t.Fatal("round trip")
	}
	if Now() <= 0 || NowData().IsZero() {
		t.Fatal("now")
	}
	if _, err := AgoUnixMilli("3q"); err == nil {
		t.Fatal("unknown unit")
	}
	for _, u := range []string{"1h", "1d", "1w", "1m", "1y"} {
		if v, err := AgoUnixMilli(u); err != nil || v >= time.Now().UnixMilli() {
			t.Errorf("%s: %d %v", u, v, err)
		}
	}
}
