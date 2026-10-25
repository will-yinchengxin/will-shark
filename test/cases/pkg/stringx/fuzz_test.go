package stringx

import (
	"testing"
	"unicode/utf8"

	"willshark/pkg/netx"
	"willshark/pkg/timex"
)

func FuzzStringHelpers(f *testing.F) {
	for _, s := range []string{"7d", "10.0.0.1~10.0.0.2", "10.0.0.0/8", "Ünicode", "2024-03-02 00:00:00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = timex.AgoUnixMilli(s)
		_ = netx.ValidateIP(s)
		_ = timex.ParseTimeStrToInt(s)
		_ = timex.GetTimeStrSlice(s, s)
		if utf8.ValidString(s) && !utf8.ValidString(LowerFirst(s)) {
			t.Fatalf("LowerFirst(%q) produced invalid UTF-8", s)
		}
	})
}
