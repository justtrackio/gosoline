package funk

import (
	"math"
)

func Map[S ~[]T1, T1, T2 any](original S, op func(T1) T2) []T2 {
	result := make([]T2, 0, len(original))

	for _, item := range original {
		result = append(result, op(item))
	}

	return result
}

func Filter[S ~[]T, T any](original S, pred func(T) bool) []T {
	if len(original) == 0 {
		return []T{}
	}

	result := make([]T, 0, len(original))

	for _, item := range original {
		if pred(item) {
			result = append(result, item)
		}
	}

	// minimize slice suffix padding
	if cap(result)-len(result) > len(result)/10 {
		result = append(make([]T, 0, len(result)), result[:len(result)]...)
	}

	return result
}

func Reduce[S ~[]T1, T1, T2 any](original S, op func(T2, T1, int) T2, init T2) T2 {
	result := init

	for i, item := range original {
		result = op(result, item, i)
	}

	return result
}

func ToMap[S ~[]T, T, V any, K comparable](sl S, keyer func(T) (K, V)) map[K]V {
	out := make(map[K]V, len(sl))

	for _, item := range sl {
		k, v := keyer(item)
		out[k] = v
	}

	return out
}

func ToSet[S ~[]T, T comparable](original S) Set[T] {
	result := make(Set[T], len(original))

	for _, item := range original {
		result.Set(item)
	}

	return result
}

func FindFirst[S ~[]T, T comparable](sl S, el T) (index int, ok bool) {
	for i, item := range sl {
		if item == el {
			return i, true
		}
	}

	return
}

func FindFirstFunc[S ~[]T, T any](sl S, pred func(T) bool) (index int, ok bool) {
	for i, item := range sl {
		if pred(item) {
			return i, true
		}
	}

	return 0, false
}

func First[S ~[]T, T any](sl S) (out T, ok bool) {
	if len(sl) >= 1 {
		return sl[0], true
	}

	return
}

func Chunk[S ~[]T, T any](original S, size int) [][]T {
	if size < 1 || len(original) == 0 {
		return nil
	}

	// TODO: the last bucket does not always need size elements, most times it needs less
	bucketCount := len(original) / size
	if len(original) % size != 0 {
		bucketCount++
	}

	result := make([][]T, bucketCount)
	for i := 0; i < bucketCount; i++ {
		result[i] = make([]T, 0, size)
	}

	for i := 0; i < len(original); i++ {
		result[i/size] = append(result[i/size], original[i])
	}

	return result
}

func ChunkReduce[S ~[]T, T any](sl S, size int) (out [][]T) {
	if size < 1 || len(sl) == 0 {
		return
	}

	return Reduce(sl, func(result [][]T, item T, idx int) [][]T {
		chunkIdx := int(math.Floor(float64(idx) / float64(size)))
		if len(result) < chunkIdx+1 || result[chunkIdx] == nil {
			result = append(result, []T{})
		}

		result[chunkIdx] = append(result[chunkIdx], item)

		return result
	}, [][]T{})
}

func Reverse[S ~[]T, T any](sl S) (out S) {
	out = make(S, 0, len(sl))
	for i := len(sl) - 1; i >= 0; i-- {
		out = append(out, sl[i])
	}

	return
}

func FromSet[T comparable](s Set[T]) (out []T) {
	for k := range s {
		out = append(out, k)
	}

	return
}

func Uniq[T comparable](sl []T) (out []T) {
	return FromSet(ToSet(sl))
}

// TODO make it work for ~[]T too
func Difference[T comparable](sl1, sl2 []T) (left, right []T) {
	set1, set2 := ToSet(sl1), ToSet(sl2)

	for _, item := range sl1 {
		if !set2.Contains(item) {
			left = append(left, item)
		}
	}

	for _, item := range sl2 {
		if !set1.Contains(item) {
			right = append(right, item)
		}
	}

	return
}

func Intersect[T comparable](sl1, sl2 []T) (out []T) {
	set1, set2 := ToSet(sl1), ToSet(sl2)

	for val := range set1 {
		if set2.Contains(val) {
			out = append(out, val)
		}
	}

	return
}
