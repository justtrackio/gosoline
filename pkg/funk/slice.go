package funk

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

// CastSlice casts each element of sl from any to T.
// It returns an error when an element cannot be asserted to T.
func CastSlice[T any, I ~[]any](sl I) ([]T, error) {
	result := make([]T, len(sl))

	for i := 0; i < len(sl); i++ {
		el, ok := sl[i].(T)
		if !ok {
			return nil, fmt.Errorf("could not cast element at index %d", i)
		}

		result[i] = el
	}

	return result, nil
}

// BoxSlice returns a slice containing pointers to the values in sl.
func BoxSlice[T any](sl []T) []*T {
	return Map(sl, mdl.Box)
}

// UnboxSlice returns the values referenced by sl, using the zero value for nil pointers.
func UnboxSlice[T any](sl []*T) []T {
	return Map(sl, mdl.EmptyIfNil)
}

// Partition groups the elements of sl by the key returned from keyer.
func Partition[S ~[]T, T any, E comparable, F func(T) E](sl S, keyer F) map[E][]T {
	result := make(map[E][]T)

	for _, item := range sl {
		key := keyer(item)
		result[key] = append(result[key], item)
	}

	return result
}

// PartitionMap groups mapped values by the key returned from keyer for each input element.
func PartitionMap[S ~[]T, T, K comparable, V any, F func(T) (K, V)](inp S, keyer F) map[K][]V {
	out := make(map[K][]V)

	for _, item := range inp {
		key, val := keyer(item)
		out[key] = append(out[key], val)
	}

	return out
}

// Chunk splits sl into consecutive chunks of at most size elements.
// It returns nil for non-positive sizes and an empty slice for empty input.
func Chunk[S ~[]T, T any](sl S, size int) [][]T {
	if size <= 0 {
		return nil
	}

	if len(sl) == 0 {
		return make([][]T, 0)
	}

	chunkCount := len(sl) / size
	result := make([][]T, chunkCount)

	for i := 0; i < chunkCount; i++ {
		result[i] = make([]T, 0, size)
	}

	if rem := len(sl) % size; rem != 0 {
		result = append(result, make([]T, 0, rem))
	}

	for i := 0; i < len(sl); i++ {
		result[i/size] = append(result[i/size], sl[i])
	}

	return result
}

// ChunkIntoBuckets splits sl into chunks sized to fit the input into numOfBuckets buckets.
func ChunkIntoBuckets[S ~[]T, T any](sl S, numOfBuckets int) [][]T {
	size := int(math.Ceil(float64(len(sl)) / float64(numOfBuckets)))

	return Chunk(sl, size)
}

// Concat appends all provided slices into a new slice in argument order.
func Concat[T any](sl ...[]T) []T {
	final := make([]T, 0)

	for i := 0; i < len(sl); i++ {
		final = append(final, sl[i]...)
	}

	return final
}

// Contains reports whether in contains an element deeply equal to elem.
func Contains[T any](in []T, elem T) bool {
	equalTo := equal(elem)

	return ContainsFunc(in, equalTo)
}

// ContainsAll reports whether in contains every element from elements.
func ContainsAll[T any](in []T, elements []T) bool {
	for _, elem := range elements {
		if !Contains(in, elem) {
			return false
		}
	}

	return true
}

// ContainsFunc reports whether any element in sl satisfies pred.
func ContainsFunc[S ~[]T, T any](sl S, pred func(T) bool) bool {
	_, ok := FindFirstFunc(sl, pred)

	return ok
}

// Difference returns elements only present in left and elements only present in right.
// Example: Difference([]int{1, 2, 3}, []int{2, 3, 4}) returns []int{1}, []int{4}.
func Difference[S ~[]T, T comparable](left, right S) (inLeft, inRight []T) {
	set1, set2 := SliceToSet(left), SliceToSet(right)

	inLeft = make([]T, 0, len(set1))
	inRight = make([]T, 0, len(set2))

	for _, item := range left {
		if !set2.Contains(item) {
			inLeft = append(inLeft, item)
		}
	}

	for _, item := range right {
		if !set1.Contains(item) {
			inRight = append(inRight, item)
		}
	}

	return inLeft, inRight
}

// DifferenceKeyed returns keyed elements only present in left and only present in right.
// Elements are compared by their mdl.Keyed key.
func DifferenceKeyed[S1 ~[]T1, S2 ~[]T2, T1, T2 mdl.Keyed](left S1, right S2) (inLeft S1, inRight S2) {
	inLeftS, inRightS := DifferenceMaps(KeyedToMap(left), KeyedToMap(right))

	return Values(inLeftS), Values(inRightS)
}

// Filter returns the elements of sl for which pred returns true.
func Filter[S ~[]T, T any](sl S, pred func(T) bool) []T {
	if len(sl) == 0 {
		return []T{}
	}

	result := make([]T, 0, len(sl))
	for _, item := range sl {
		if pred(item) {
			result = append(result, item)
		}
	}

	return slices.Clip(result)
}

// FindFirst returns the first element in sl equal to el and whether such an element was found.
func FindFirst[S ~[]T, T comparable](sl S, el T) (out T, ok bool) {
	for _, item := range sl {
		if item == el {
			return item, true
		}
	}

	return
}

// FindFirstFunc returns the first element in sl satisfying pred and whether such an element was found.
func FindFirstFunc[S ~[]T, T any](sl S, pred func(T) bool) (out T, ok bool) {
	for _, item := range sl {
		if pred(item) {
			return item, true
		}
	}

	return
}

// First returns the first element of sl and whether sl was non-empty.
func First[S ~[]T, T any](sl S) (out T, ok bool) {
	if len(sl) >= 1 {
		return sl[0], true
	}

	return
}

// Flatten concatenates a slice of slices into a single slice.
func Flatten[S ~[]T, T any](sl []S) []T {
	result := make([]T, 0)
	for _, items := range sl {
		result = append(result, items...)
	}

	return result
}

// Index returns the first index of an element deeply equal to e, or -1 if none exists.
func Index[T any](sl []T, e T) int {
	for i, v := range sl {
		equalTo := equal(e)

		if equalTo(v) {
			return i
		}
	}

	return -1
}

// Intersect returns the unique elements contained in both slices.
func Intersect[S ~[]T, T comparable](sl1, sl2 S) []T {
	set2 := SliceToSet(sl2)
	result := make(Set[T])

	for _, item := range sl1 {
		if set2.Contains(item) {
			result.Add(item)
		}
	}

	return Keys(result)
}

// IntersectKeyed returns keyed elements from s1 whose keys are also present in s2.
func IntersectKeyed[S ~[]T, T mdl.Keyed](s1, s2 S) S {
	return Values(IntersectMaps(KeyedToMap(s1), KeyedToMap(s2)))
}

// KeyedToMap indexes keyed elements by their mdl.Keyed key.
func KeyedToMap[S ~[]T, T mdl.Keyed](sl S) map[string]T {
	out := make(map[string]T, len(sl))

	for _, item := range sl {
		out[item.GetKey()] = item
	}

	return out
}

// Last returns the last element of sl, or the zero value of T when sl is empty.
func Last[T any](sl []T) T {
	if len(sl) == 0 {
		var ret T

		return ret
	}

	return sl[len(sl)-1]
}

// Map applies op to each element of sl and returns the mapped values.
func Map[S ~[]T1, T1, T2 any, F func(T1) T2](sl S, op F) []T2 {
	result := make([]T2, len(sl))
	for idx, item := range sl {
		result[idx] = op(item)
	}

	return result
}

// Reduce folds sl from left to right, passing the accumulator, element, and index to op.
func Reduce[S ~[]T1, T1, T2 any](sl S, op func(T2, T1, int) T2, init T2) T2 {
	result := init

	for idx, item := range sl {
		result = op(result, item, idx)
	}

	return result
}

// SliceToMap builds a map from sl using keyer to derive each key and value.
func SliceToMap[S ~[]T, T, V any, K comparable](sl S, keyer func(T) (K, V)) map[K]V {
	out := make(map[K]V, len(sl))

	for _, item := range sl {
		k, v := keyer(item)
		out[k] = v
	}

	return out
}

// SliceToSet returns a set containing the unique elements of sl.
func SliceToSet[S ~[]T, T comparable](sl S) Set[T] {
	return NewSet[T](sl...)
}

// Repeat returns a slice containing el repeated times times.
// It returns nil for negative times and an empty slice for zero times.
func Repeat[T any](el T, times int) []T {
	if times < 0 {
		return nil
	}

	if times == 0 {
		return []T{}
	}

	result := make([]T, times)

	for i := 0; i < times; i++ {
		result[i] = el
	}

	return result
}

// Reverse returns a new slice with the elements of sl in reverse order.
func Reverse[S ~[]T, T any](sl S) S {
	out := make(S, len(sl))
	for i, j := 0, len(sl)-1; i < len(sl) && j >= 0; i, j = i+1, j-1 {
		out[i] = sl[j]
	}

	return out
}

// Tail returns sl without its first element.
// For slices with fewer than two elements, it returns sl unchanged.
func Tail[T any](sl []T) []T {
	if len(sl) < 2 {
		return sl
	}

	return sl[1:]
}

// Uniq returns sl without duplicate elements, preserving the first occurrence of each element.
func Uniq[S ~[]T, T comparable](sl S) S {
	set := make(Set[T], len(sl))
	res := make(S, 0)

	for _, e := range sl {
		if set.Contains(e) {
			continue
		}

		set.Add(e)
		res = append(res, e)
	}

	return res
}

// UniqFunc returns sl without duplicate keys produced by fn, preserving the first element for each key.
func UniqFunc[S ~[]T, T any, K comparable](sl S, fn func(T) K) []T {
	keys := make(Set[K], len(sl))
	uniq := make([]T, 0)

	for _, item := range sl {
		key := fn(item)

		if keys.Contains(key) {
			continue
		}

		keys.Add(key)
		uniq = append(uniq, item)
	}

	return uniq
}

// UniqByType returns sl without duplicate dynamic types, preserving the first element for each type.
func UniqByType[S ~[]T, T any](sl S) S {
	types := map[reflect.Type]bool{}

	return Filter(sl, func(a T) bool {
		t := reflect.TypeOf(a)
		if types[t] {
			return false
		}

		types[t] = true

		return true
	})
}

func equal[T any](expected T) func(actualValue T) bool {
	return func(actualValue T) bool {
		return reflect.DeepEqual(actualValue, expected)
	}
}

// Any reports whether any element in inp satisfies pred.
func Any[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	_, ok := FindFirstFunc(inp, pred)

	return ok
}

// None reports whether no element in inp satisfies pred.
func None[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	return !Any(inp, pred)
}

// All reports whether every element in inp satisfies pred.
func All[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	return None(inp, func(i T) bool { return !pred(i) })
}

// Empty reports whether inp has no elements.
func Empty[S ~[]T, T any](inp S) bool {
	return len(inp) == 0
}

// NotEmpty reports whether inp has at least one element.
func NotEmpty[S ~[]T, T any](inp S) bool {
	return len(inp) > 0
}

// NilIfEmpty returns nil for empty slices and inp otherwise.
func NilIfEmpty[S ~[]T, T any](inp S) S {
	if Empty(inp) {
		var nilSlice S

		return nilSlice
	}

	return inp
}
