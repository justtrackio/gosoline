package funk

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// MergeMaps merges zero or more maps into one combined map. Elements from later arguments overwrite elements
// from earlier arguments in the case of collisions. If given a single argument, MergeMaps produces a swallow
// copy of the input.
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

// MergeMapsWith is similar to MergeMaps, but uses a function to combine elements from keys present in multiple
// input maps.
func MergeMapsWith[K comparable, V any, M ~map[K]V](combine func(V, V) V, m ...M) M {
	var length int
	for _, item := range m {
		length += len(item)
	}

	out := make(M, length)
	for _, item := range m {
		for k, v := range item {
			if existing, ok := out[k]; ok {
				out[k] = combine(existing, v)
			} else {
				out[k] = v
			}
		}
	}

	return out
}

// IntersectMaps produces the intersection of at least two maps. The resulting map contains all elements from
// the first map with keys present in all input maps.
func IntersectMaps[K comparable, V any, M ~map[K]V](m1, m2 M, ms ...M) M {
	result := make(M)

	for k, v := range m1 {
		if _, ok := m2[k]; ok && All(ms, func(m3 M) bool {
			_, ok := m3[k]

			return ok
		}) {
			result[k] = v
		}
	}

	return result
}

// IntersectMapsWith is similar to IntersectMaps, but selects the values based on a combination function.
func IntersectMapsWith[K comparable, V any, M ~map[K]V](combine func(V, V) V, m1, m2 M, ms ...M) M {
	result := make(M)

	for k, v1 := range m1 {
		if v2, ok := m2[k]; ok && All(ms, func(m3 M) bool {
			_, ok := m3[k]

			return ok
		}) {
			result[k] = combine(v1, v2)
			for _, m3 := range ms {
				result[k] = combine(result[k], m3[k])
			}
		}
	}

	return result
}

// DifferenceMaps splits two maps into two new maps. The first result contains only the part of the left input
// map not present in the right input map while the second result contains the part from the right input map
// without keys in the left input map.
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

// Keys returns the keys from a map as a slice. The order of the result will be undefined.
func Keys[K comparable, V any, M ~map[K]V](m M) []K {
	keys := make([]K, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

// Values returns the values from a map as a slice. The order of the result will be undefined.
func Values[K comparable, V any, M ~map[K]V](m M) []V {
	values := make([]V, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}

func MapFilter[K comparable, V any, M ~map[K]V](m M, f func(key K, value V) bool) map[K]V {
	filteredMap := map[K]V{}

	for k, v := range m {
		if f(k, v) {
			filteredMap[k] = v
		}
	}

	return filteredMap
}

// MapKeys applies a function to every key from a map. If the function maps two keys to the same new value,
// the result is undefined (one of the values will be randomly chosen).
func MapKeys[K1 comparable, K2 comparable, V any, M1 ~map[K1]V](m M1, f func(key K1) K2) map[K2]V {
	r := make(map[K2]V, len(m))

	for k, v := range m {
		r[f(k)] = v
	}

	return r
}

// MapKeysWith is similar to MapKeys, but uses a combination function to resolve conflicts when mapping two keys
// to the same new key value. The order of the arguments of the combination function is randomly chosen and not
// defined.
func MapKeysWith[K1 comparable, K2 comparable, V any, M1 ~map[K1]V](m M1, f func(key K1) K2, combine func(V, V) V) map[K2]V {
	r := make(map[K2]V, len(m))

	for k, v := range m {
		newKey := f(k)
		if existing, ok := r[newKey]; ok {
			r[newKey] = combine(existing, v)
		} else {
			r[newKey] = v
		}
	}

	return r
}

// MapValues applies a function to all values of a map.
func MapValues[K comparable, V1, V2 any, M1 ~map[K]V1](m M1, f func(value V1) V2) map[K]V2 {
	r := make(map[K]V2, len(m))

	for k, v := range m {
		r[k] = f(v)
	}

	return r
}

// PopulateMap creates a new map from a value and a list of keys. All keys will be mapped to the given value.
func PopulateMap[V any, K comparable](value V, keys ...K) map[K]V {
	result := make(map[K]V, len(keys))

	for _, key := range keys {
		result[key] = value
	}

	return result
}

// PopulateMapWith creates a new map from a function and a list of keys by applying the function to each key.
func PopulateMapWith[T any, K comparable](generator func(K) T, keys ...K) map[K]T {
	result := make(map[K]T, len(keys))

	for _, key := range keys {
		result[key] = generator(key)
	}

	return result
}

// RangeSorted returns an iterator that yields all key/value pairs from the given map in order sorted by key.
func RangeSorted[K cmp.Ordered, V any](m map[K]V) iter.Seq2[K, V] {
	keys := slices.Sorted[K](maps.Keys(m))

	return func(yield func(K, V) bool) {
		for _, k := range keys {
			if !yield(k, m[k]) {
				return
			}
		}
	}
}
