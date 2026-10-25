package utils

import (
	"testing"
	"unicode/utf8"

	"willshark/utils/strings"
)

func FuzzStringHelpers(f *testing.F) {
	for _, s := range []string{"7d", "10.0.0.1~10.0.0.2", "10.0.0.0/8", "Ünicode", "2024-03-02 00:00:00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = AgoUnixMilli(s)
		_ = ValidateIP(s)
		_ = ParseTimeStrToInt(s)
		_ = GetTimeStrSlice(s, s)
		if utf8.ValidString(s) && !utf8.ValidString(strings.LowerFirst(s)) {
			t.Fatalf("LowerFirst(%q) produced invalid UTF-8", s)
		}
	})
}
