package atomic

import "testing"

func TestBoolean(t *testing.T) {
	var b Boolean
	if b.Get() {
		t.Fatal("zero value must be false")
	}
	b.Set(true)
	if !b.Get() {
		t.Fatal("set true")
	}
	b.Set(false)
	if b.Get() {
		t.Fatal("set false")
	}
}
