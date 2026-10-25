package hash

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestConsistentHashGetAndRemove(t *testing.T) {
	h := NewConsistentHash()
	for i := range 3 {
		h.Add(fmt.Sprintf("node-%d", i))
	}
	n, ok := h.Get("key-1")
	if !ok {
		t.Fatal("get failed")
	}
	if again, _ := h.Get("key-1"); again != n {
		t.Fatal("mapping must be stable")
	}
	h.Remove(n)
	if moved, _ := h.Get("key-1"); moved == n {
		t.Fatal("removed node must not be returned")
	}
}

// 并发 Add 同一节点：虚拟节点数量必须保持为 replicas，不能重复插入
func TestConcurrentAddSameNode(t *testing.T) {
	h := NewConsistentHash()
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { h.Add("node") })
	}
	wg.Wait()
	if len(h.keys) != minReplicas {
		t.Fatalf("keys = %d, want %d", len(h.keys), minReplicas)
	}
}

func TestAddWithWeightAndCustomHash(t *testing.T) {
	h := NewCustomConsistentHash(1, nil) // replicas 下限 100，nil 使用默认哈希
	h.AddWithWeight("half", 50)
	if len(h.keys) != minReplicas/2 {
		t.Fatalf("keys = %d", len(h.keys))
	}
	// 所有输入哈希相同：同一环位置上有多个节点，按 innerRepr 再选一次
	same := NewCustomConsistentHash(minReplicas, func([]byte) uint64 { return 1 })
	same.Add("a")
	same.Add("b")
	if n, ok := same.Get("k"); !ok || (n != "a" && n != "b") {
		t.Fatalf("collision get = %v %v", n, ok)
	}
	same.Remove("a")
	if n, _ := same.Get("k"); n != "b" {
		t.Fatalf("after remove = %v", n)
	}
	if _, ok := NewConsistentHash().Get("k"); ok {
		t.Fatal("empty ring")
	}
}

type stringer struct{}

func (stringer) String() string { return "S" }

func TestReprOfValue(t *testing.T) {
	n := 5
	var nilPtr *int
	cases := map[string]any{
		"": nil, "true": true, "e": errors.New("e"), "1.5": 1.5, "2.5": float32(2.5), "S": stringer{},
		"1": int8(1), "2": int16(2), "3": int32(3), "4": int64(4), "6": uint(6), "7": uint8(7),
		"8": uint16(8), "9": uint32(9), "10": uint64(10), "s": "s", "b": []byte("b"), "5": &n, "<nil>": nilPtr,
		"[1 2]": []int{1, 2},
	}
	for want, in := range cases {
		if got := ReprOfValue(in); got != want {
			t.Errorf("ReprOfValue(%#v) = %q, want %q", in, got, want)
		}
	}
	if Md5Hex([]byte("a")) != "0cc175b9c0f1b6a831c399e269772661" {
		t.Fatal("md5")
	}
}

func TestNilNode(t *testing.T) {
	h := NewConsistentHash()
	h.Add(nil)
	if n, ok := h.Get("k"); !ok || n != nil {
		t.Fatalf("nil node: %v %v", n, ok)
	}
}
