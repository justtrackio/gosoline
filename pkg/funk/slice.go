package funk

import (
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"golang.org/x/exp/maps"
)

// CastSlice casts a []any slice to the given type.
// The parameter sliceType is required to correctly infer the target type.
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

func Partition[S ~[]T, T any, E comparable, F func(T) E](sl S, keyer F) map[E][]T {
	result := make(map[E][]T)

	for _, item := range sl {
		key := keyer(item)
		result[key] = append(result[key], item)
	}

	return result
}

func PartitionMap[S ~[]T, T, K comparable, V any, F func(T) (K, V)](inp S, keyer F) map[K][]V {
	out := make(map[K][]V)

	for _, item := range inp {
		key, val := keyer(item)
		out[key] = append(out[key], val)
	}

	return out
}

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

func ChunkIntoBuckets[S ~[]T, T any](sl S, numOfBuckets int) [][]T {
	size := int(math.Ceil(float64(len(sl)) / float64(numOfBuckets)))

	return Chunk(sl, size)
}

func Contains[T any](in []T, elem T) bool {
	equalTo := equal(elem)

	return ContainsFunc(in, equalTo)
}

func ContainsAll[T any](in []T, elements []T) bool {
	for _, elem := range elements {
		if !Contains(in, elem) {
			return false
		}
	}

	return true
}

func ContainsFunc[S ~[]T, T any](sl S, pred func(T) bool) bool {
	_, ok := FindFirstFunc(sl, pred)

	return ok
}

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

func DifferenceKeyed[S1 ~[]T1, S2 ~[]T2, T1, T2 mdl.Keyed](left S1, right S2) (inLeft S1, inRight S2) {
	inLeftS, inRightS := DifferenceMaps(KeyedToMap(left), KeyedToMap(right))

	return maps.Values(inLeftS), maps.Values(inRightS)
}

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

func Flatten[S ~[]T, T any](sl []S) []T {
	result := make([]T, 0)
	for _, items := range sl {
		result = append(result, items...)
	}

	return result
}

func Index[T any](sl []T, e T) int {
	for i, v := range sl {
		equalTo := equal(e)

		if equalTo(v) {
			return i
		}
	}

	return -1
}

func Intersect[S ~[]T, T comparable](sl1, sl2 S) []T {
	set2 := SliceToSet(sl2)
	result := make(Set[T])

	for _, item := range sl1 {
		if set2.Contains(item) {
			result.Set(item)
		}
	}

	return maps.Keys(result)
}

func IntersectKeyed[S ~[]T, T mdl.Keyed](s1, s2 S) S {
	return maps.Values(IntersectMaps(KeyedToMap(s1), KeyedToMap(s2)))
}

func KeyedToMap[S ~[]T, T mdl.Keyed](sl S) map[string]T {
	out := make(map[string]T, len(sl))

	for _, item := range sl {
		out[item.GetKey()] = item
	}

	return out
}

func Last[T any](sl []T) T {
	if len(sl) == 0 {
		var ret T

		return ret
	}

	return sl[len(sl)-1]
}

func Map[S ~[]T1, T1, T2 any, F func(T1) T2](sl S, op F) []T2 {
	result := make([]T2, len(sl))
	for idx, item := range sl {
		result[idx] = op(item)
	}

	return result
}

func Reduce[S ~[]T1, T1, T2 any](sl S, op func(T2, T1, int) T2, init T2) T2 {
	result := init

	for idx, item := range sl {
		result = op(result, item, idx)
	}

	return result
}

func SliceToMap[S ~[]T, T, V any, K comparable](sl S, keyer func(T) (K, V)) map[K]V {
	out := make(map[K]V, len(sl))

	for _, item := range sl {
		k, v := keyer(item)
		out[k] = v
	}

	return out
}

func SliceToSet[S ~[]T, T comparable](sl S) Set[T] {
	result := make(Set[T], len(sl))

	for _, item := range sl {
		result.Set(item)
	}

	return result
}

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

func Reverse[S ~[]T, T any](sl S) S {
	out := make(S, len(sl))
	for i, j := 0, len(sl)-1; i < len(sl) && j >= 0; i, j = i+1, j-1 {
		out[i] = sl[j]
	}

	return out
}

func Tail[T any](sl []T) []T {
	if len(sl) < 2 {
		return sl
	}

	return sl[1:]
}

func Uniq[S ~[]T, T comparable](sl S) S {
	set := make(Set[T], len(sl))
	res := make(S, 0)

	for _, e := range sl {
		if set.Contains(e) {
			continue
		}

		set.Set(e)
		res = append(res, e)
	}

	return res
}

func UniqFunc[S ~[]T, T any, K comparable](sl S, fn func(T) K) []T {
	keys := make(Set[K], len(sl))
	uniq := make([]T, 0)

	for _, item := range sl {
		key := fn(item)

		if keys.Contains(key) {
			continue
		}

		keys.Set(key)
		uniq = append(uniq, item)
	}

	return uniq
}

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

func Any[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	_, ok := FindFirstFunc(inp, pred)

	return ok
}

func None[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	return !Any(inp, pred)
}

func All[S ~[]T, T any, F func(T) bool](inp S, pred F) bool {
	return None(inp, func(i T) bool { return !pred(i) })
}

func Empty[S ~[]T, T any](inp S) bool {
	return len(inp) == 0
}

func NotEmpty[S ~[]T, T any](inp S) bool {
	return len(inp) > 0
}
