package structx

import (
	"testing"
)

func TestCopyStructFields(t *testing.T) {
	type src struct {
		Name string
		Age  int
	}
	type dst struct{ Name string }
	var d dst
	if err := CopyStructFields(&d, src{Name: "a", Age: 1}); err != nil || d.Name != "a" {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}
