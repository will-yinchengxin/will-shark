package randx

import "testing"

func TestRandn(t *testing.T) {
	if s := Randn(16); len(s) != 16 {
		t.Fatalf("len = %d", len(s))
	}
	if a, b := RandID(), RandID(); a == b {
		t.Fatal("RandID must be random")
	}
}

func TestSeedDeterministic(t *testing.T) {
	Seed(42)
	a := Rand()
	Seed(42)
	if b := Rand(); a != b || len(a) != defaultRandLen {
		t.Fatalf("a=%q b=%q", a, b)
	}
}
