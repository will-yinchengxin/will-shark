package slicex

import (
	"slices"
	"testing"
)

func TestInArray(t *testing.T) {
	if ok, i := InArray(2, []int{1, 2}); !ok || i != 1 {
		t.Fatalf("ok=%v i=%d", ok, i)
	}
	if ok, i := InArray(1, nil); ok || i != -1 {
		t.Fatal("nil haystack must return not found (was a panic)")
	}
	if ok, _ := InArray(1, 5); ok {
		t.Fatal("non-slice haystack")
	}
}

func TestSliceMergeDoesNotAlias(t *testing.T) {
	a := make([]int, 2, 10)
	b := SliceMergeInt(a, []int{7})
	_ = SliceMergeInt(a, []int{9})
	if b[2] != 7 {
		t.Fatalf("merge result was overwritten through a shared backing array: %v", b)
	}
}

func TestFindDiff(t *testing.T) {
	add, del := FindDiff([]int64{1, 2, 3}, []int64{2, 3, 4})
	slices.Sort(add)
	slices.Sort(del)
	if !slices.Equal(add, []int64{1}) || !slices.Equal(del, []int64{4}) {
		t.Fatalf("add=%v del=%v", add, del)
	}
}

func TestSliceHelpers(t *testing.T) {
	if !SliceContains([]interface{}{1, "a"}, "a") || !SliceContainsInt([]int{1}, 1) ||
		!SliceContainsInt64([]int64{2}, 2) || !SliceContainsString([]string{"x"}, "x") || SliceContainsString(nil, "x") {
		t.Fatal("contains")
	}
	u := SliceUniqueInt64([]int64{3, 1, 3})
	slices.Sort(u)
	if !slices.Equal(u, []int64{1, 3}) || len(SliceUniqueInt(nil)) != 0 || len(SliceUniqueString([]string{"a", "a"})) != 1 || len(SliceUniqueInt([]int{1, 1, 2})) != 2 {
		t.Fatal("unique")
	}
	if SliceSumInt([]int{1, 2}) != 3 || SliceSumInt64([]int64{1, 2}) != 3 || SliceSumFloat64([]float64{0.5, 0.5}) != 1 {
		t.Fatal("sum")
	}
	if len(SliceMerge([]interface{}{1}, []interface{}{2})) != 2 || len(SliceMergeString(nil, []string{"a"})) != 1 || len(SliceMergeInt64(nil, nil)) != 0 {
		t.Fatal("merge")
	}
}
