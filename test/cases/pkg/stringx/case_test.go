package stringx

import "testing"

func TestLowerFirst(t *testing.T) {
	for in, want := range map[string]string{"": "", "Abc": "abc", "Ünicode": "ünicode", "中文": "中文", "A": "a"} {
		if got := LowerFirst(in); got != want {
			t.Errorf("LowerFirst(%q) = %q, want %q", in, got, want)
		}
	}
}
