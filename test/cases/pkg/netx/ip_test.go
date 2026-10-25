package netx

import (
	"testing"
)

func TestValidateIP(t *testing.T) {
	cases := map[string]bool{
		"10.0.0.1": true, "::1": true, "10.0.0.0/8": true, "10.0.0.1~10.0.0.9": true,
		"10.0.0.0/33": false, "a~b": false, "1.1.1.1~": false, "x": false, "": false,
	}
	for in, want := range cases {
		if got := ValidateIP(in); got != want {
			t.Errorf("ValidateIP(%q) = %v", in, got)
		}
	}
}
