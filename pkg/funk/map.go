package funk

func MergeMaps[K comparable, V any, M ~map[K]V](m ...M) M {
	var length int
	for _, item := range m {
		length += len(item)
	}

	out := make(M, length)
	for _, item := range m {
		for k, v := range item {
			out[k] = v
		}
	}

	return out
}

func IntersectMaps[K comparable, V any, M ~map[K]V](m1, m2 M) M {
	result := make(M)

	for k, v := range m1 {
		if _, ok := m2[k]; ok {
			result[k] = v
		}
	}

	return result
}

func DifferenceMaps[K comparable, V1, V2 any, M1 ~map[K]V1, M2 ~map[K]V2](left M1, right M2) (inLeft M1, inRight M2) {
	inLeft, inRight = make(M1, len(left)), make(M2, len(right))

	for k, v := range left {
		if _, ok := right[k]; !ok {
			inLeft[k] = v
		}
	}

	for k, v := range right {
		if _, ok := left[k]; !ok {
			inRight[k] = v
		}
	}

	return inLeft, inRight
}
