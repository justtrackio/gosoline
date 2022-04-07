package funk

import (
	"math"
)

func Map[S ~[]T1, T1, T2 any](sl S, op func(T1) T2) (out []T2) {
	for _, item := range sl {
		out = append(out, op(item))
	}

	return
}

func Filter[S ~[]T, T any](sl S, pred func(T) bool) (out []T) {
	for _, item := range sl {
		if pred(item) {
			out = append(out, item)
		}
	}

	return
}

func Reduce[S ~[]T1, T1, T2 any](sl S, op func(T2, T1, int) T2, init T2) (out T2) {
	out = init

	for idx, item := range sl {
		out = op(out, item, idx)
	}

	return
}

func ToMap[S ~[]T, T, V any, K comparable](sl S, keyer func(T) (K, V)) map[K]V {
	out := make(map[K]V, len(sl))

	for _, item := range sl {
		k, v := keyer(item)
		out[k] = v
	}

	return out
}

func ToSet[S ~[]T, T comparable](sl S) Set[T] {
	out := make(Set[T], 0)
	for _, item := range sl {
		out.Set(item)
	}

	return out
}

func FindFirst[S ~[]T, T comparable](sl S, el T) (out T, ok bool) {
	for _, item := range sl {
		if item == el {
			return item, true
		}
	}

	return
}

func FindFirstFunc[S ~[]T, T any](sl S, pred func(T) bool) (out T, ok bool) {
	for _, item := range sl {
		if pred(item) {
			return item, true
		}
	}

	return
}

func First[S ~[]T, T any](sl S) (out T, ok bool) {
	if len(sl) >= 1 {
		return sl[0], true
	}

	return
}

func Chunk[S ~[]T, T any](sl S, size int) (out [][]T) {
	if size < 1 || len(sl) == 0 {
		return
	}

	var chunk []T
	length := len(sl)

	for i := 0; i < length; i++ {
		if i%size == 0 || i == 0 {
			if len(chunk) > 0 {
				out = append(out, chunk)
			}

			chunk = []T{}
		}

		chunk = append(chunk, sl[i])

		if i == length-1 {
			out = append(out, chunk)
		}
	}

	return
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
