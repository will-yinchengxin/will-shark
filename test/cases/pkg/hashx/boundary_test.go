package hashx

import "testing"

func TestRemoveWeightedNodePreservesCollidingNode(t *testing.T) {
	h := NewCustomConsistentHash(100, func([]byte) uint64 { return 1 })
	h.AddWithReplicas("one", 1)
	h.AddWithReplicas("half", 50)
	h.Remove("half")
	if n, ok := h.Get("key"); !ok || n != "one" {
		t.Fatalf("remaining node=%v ok=%v", n, ok)
	}
}
