package utils

// FindDiff 用于穿梭框的算法应用
func FindDiff(slice1, slice2 []int64) (addIDs []int64, delIDs []int64) {
	m1 := make(map[int64]bool)
	m2 := make(map[int64]bool)

	for _, s := range slice1 {
		m1[s] = true
	}
	for _, s := range slice2 {
		m2[s] = true
	}

	diff1 := make([]int64, 0)
	for s := range m1 {
		if !m2[s] {
			diff1 = append(diff1, s)
		}
	}

	diff2 := make([]int64, 0)
	for s := range m2 {
		if !m1[s] {
			diff2 = append(diff2, s)
		}
	}

	return diff1, diff2
}
