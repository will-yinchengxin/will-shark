package randx

import (
	"testing"
)

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
