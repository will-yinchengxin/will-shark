// Package slicex 切片工具。新代码优先使用标准库 slices / maps。
package slicex

import (
	"reflect"
	"slices"
)

func InArray(needle interface{}, haystack interface{}) (exists bool, index int) {
	exists = false
	index = -1
	if haystack == nil {
		return
	}

	switch reflect.TypeOf(haystack).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(haystack)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(needle, s.Index(i).Interface()) {
				index = i
				exists = true
				return
			}
		}
	}

	return
}

func SliceContains(sl []interface{}, v interface{}) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

func SliceContainsInt(sl []int, v int) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

func SliceContainsInt64(sl []int64, v int64) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

func SliceContainsString(sl []string, v string) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

// SliceMerge merges interface slices to one slice.
func SliceMerge(slice1, slice2 []interface{}) (c []interface{}) {
	c = slices.Concat(slice1, slice2) // 不复用 slice1 的底层数组，避免修改调用方数据
	return
}

func SliceMergeInt(slice1, slice2 []int) (c []int) {
	c = slices.Concat(slice1, slice2) // 不复用 slice1 的底层数组，避免修改调用方数据
	return
}

func SliceMergeInt64(slice1, slice2 []int64) (c []int64) {
	c = slices.Concat(slice1, slice2) // 不复用 slice1 的底层数组，避免修改调用方数据
	return
}

func SliceMergeString(slice1, slice2 []string) (c []string) {
	c = slices.Concat(slice1, slice2) // 不复用 slice1 的底层数组，避免修改调用方数据
	return
}

func SliceUniqueInt64(s []int64) []int64 {
	size := len(s)
	if size == 0 {
		return []int64{}
	}

	m := make(map[int64]bool)
	for i := 0; i < size; i++ {
		m[s[i]] = true
	}

	realLen := len(m)
	ret := make([]int64, realLen)

	idx := 0
	for key := range m {
		ret[idx] = key
		idx++
	}

	return ret
}

func SliceUniqueInt(s []int) []int {
	size := len(s)
	if size == 0 {
		return []int{}
	}

	m := make(map[int]bool)
	for i := 0; i < size; i++ {
		m[s[i]] = true
	}

	realLen := len(m)
	ret := make([]int, realLen)

	idx := 0
	for key := range m {
		ret[idx] = key
		idx++
	}

	return ret
}

func SliceUniqueString(s []string) []string {
	size := len(s)
	if size == 0 {
		return []string{}
	}

	m := make(map[string]bool)
	for i := 0; i < size; i++ {
		m[s[i]] = true
	}

	realLen := len(m)
	ret := make([]string, realLen)

	idx := 0
	for key := range m {
		ret[idx] = key
		idx++
	}

	return ret
}

func SliceSumInt64(intslice []int64) (sum int64) {
	for _, v := range intslice {
		sum += v
	}
	return
}

func SliceSumInt(intslice []int) (sum int) {
	for _, v := range intslice {
		sum += v
	}
	return
}

func SliceSumFloat64(intslice []float64) (sum float64) {
	for _, v := range intslice {
		sum += v
	}
	return
}
