package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
)

func TestMergeMapsNoArgs(t *testing.T) {
	result := funk.MergeMaps[int, string, map[int]string]()
	expected := map[int]string{}
	assert.Equal(t, expected, result)
}

func TestMergeMapsSingle(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
	}
	result := funk.MergeMaps(m1)
	assert.Equal(t, m1, result)

	m1[3] = "three"
	expected := map[int]string{
		1: "one",
		2: "two",
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsMultiple(t *testing.T) {
	m1 := map[int]string{
		1: "one",
	}
	m2 := map[int]string{
		2: "two",
	}
	result := funk.MergeMaps(m1, m2)
	expected := map[int]string{
		1: "one",
		2: "two",
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsCollision(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
	}
	m2 := map[int]string{
		2: "dos",
		3: "three",
	}
	result := funk.MergeMaps(m1, m2)
	expected := map[int]string{
		1: "one",
		2: "dos",
		3: "three",
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsNil(t *testing.T) {
	var m1 map[int]string // nil map
	m2 := map[int]string{
		1: "one",
	}
	result := funk.MergeMaps(m1, m2)
	expected := map[int]string{
		1: "one",
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsAllNil(t *testing.T) {
	var m1 map[int]string
	var m2 map[int]string
	result := funk.MergeMaps(m1, m2)
	expected := map[int]string{}
	assert.Equal(t, expected, result)
}

func TestMergeMapsStringKeys(t *testing.T) {
	m1 := map[string]int{
		"a": 1,
		"b": 2,
	}
	m2 := map[string]int{
		"b": 20,
		"c": 3,
	}
	m3 := map[string]int{
		"c": 4,
		"d": 33,
	}
	result := funk.MergeMaps(m1, m2, m3)
	expected := map[string]int{
		"a": 1,
		"b": 20,
		"c": 4,
		"d": 33,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithNoArgs(t *testing.T) {
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith[int, int, map[int]int](combine)
	expected := map[int]int{}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithSingle(t *testing.T) {
	m1 := map[int]int{
		1: 100,
		2: 200,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1)

	m1[3] = 300

	expected := map[int]int{
		1: 100,
		2: 200,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithMultipleDistinct(t *testing.T) {
	m1 := map[int]int{
		1: 100,
	}
	m2 := map[int]int{
		2: 200,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2)
	expected := map[int]int{
		1: 100,
		2: 200,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithCollision(t *testing.T) {
	m1 := map[int]int{
		1: 10,
		2: 20,
	}
	m2 := map[int]int{
		2: 30,
		3: 40,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2)
	expected := map[int]int{
		1: 10,
		2: 20 + 30,
		3: 40,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithMultipleCollision(t *testing.T) {
	m1 := map[int]int{
		1: 1,
		2: 2,
	}
	m2 := map[int]int{
		1: 10,
		3: 3,
	}
	m3 := map[int]int{
		1: 100,
		2: 20,
		3: 30,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2, m3)
	expected := map[int]int{
		1: (1 + 10) + 100,
		2: 2 + 20,
		3: 3 + 30,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithNil(t *testing.T) {
	var m1 map[int]int // nil map
	m2 := map[int]int{
		1: 10,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2)
	expected := map[int]int{
		1: 10,
	}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithAllNil(t *testing.T) {
	var m1 map[int]int
	var m2 map[int]int
	combine := func(a, b int) int { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2)
	expected := map[int]int{}
	assert.Equal(t, expected, result)
}

func TestMergeMapsWithStringValues(t *testing.T) {
	m1 := map[string]string{
		"a": "hello",
		"b": "world",
	}
	m2 := map[string]string{
		"a": " there",
		"c": "!",
	}
	combine := func(a, b string) string { return a + b }
	result := funk.MergeMapsWith(combine, m1, m2)
	expected := map[string]string{
		"a": "hello there",
		"b": "world",
		"c": "!",
	}
	assert.Equal(t, expected, result)
}

func TestIntersectMapsTwoMaps(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	m2 := map[int]string{
		2: "dos",
		3: "tres",
		4: "cuatro",
	}

	result := funk.IntersectMaps(m1, m2)
	expected := map[int]string{
		2: "two",
		3: "three",
	}
	assert.Equal(t, expected, result, "IntersectMaps should return the intersection of m1 and m2")
}

func TestIntersectMapsMultipleAdditional(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	m2 := map[int]string{
		1: "uno",
		2: "dos",
		3: "tres",
		4: "cuatro",
	}
	m3 := map[int]string{
		1: "ein",
		3: "drei",
	}

	result := funk.IntersectMaps(m1, m2, m3)
	expected := map[int]string{
		1: "one",
		3: "three",
	}
	assert.Equal(t, expected, result, "IntersectMaps should return only keys present in all maps")
}

func TestIntersectMapsEmpty(t *testing.T) {
	// m2 is empty.
	m1 := map[int]string{
		1: "one",
		2: "two",
	}
	m2 := map[int]string{}
	result := funk.IntersectMaps(m1, m2)
	expected := map[int]string{}
	assert.Equal(t, expected, result, "IntersectMaps with an empty second map should return an empty map")

	// Alternatively, if m1 is empty, the result is also empty.
	m1 = map[int]string{}
	m2 = map[int]string{
		1: "one",
	}
	result = funk.IntersectMaps(m1, m2)
	assert.Equal(t, expected, result, "IntersectMaps with an empty first map should return an empty map")
}

func TestIntersectMapsNil(t *testing.T) {
	var m1 map[int]string // nil map
	m2 := map[int]string{
		1: "one",
	}
	result := funk.IntersectMaps(m1, m2)
	expected := map[int]string{}
	assert.Equal(t, expected, result, "IntersectMaps with a nil first map should return an empty map")

	// Now use a nil second map.
	m1 = map[int]string{
		1: "one",
		2: "two",
	}
	var m3 map[int]string // nil map as second argument
	result = funk.IntersectMaps(m1, m3)
	assert.Equal(t, expected, result, "IntersectMaps with a nil second map should return an empty map")
}

func TestIntersectMapsWithTwoMaps(t *testing.T) {
	m1 := map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}
	m2 := map[int]int{
		2: 200,
		3: 300,
		4: 400,
	}
	combine := func(a, b int) int { return a + b }

	result := funk.IntersectMapsWith(combine, m1, m2)
	expected := map[int]int{
		2: 20 + 200,
		3: 30 + 300,
	}
	assert.Equal(t, expected, result, "IntersectMapsWith should combine values for keys present in both m1 and m2")
}

func TestIntersectMapsWithAdditional(t *testing.T) {
	m1 := map[int]int{
		1: 10,
		2: 20,
		3: 30,
		4: 40,
	}
	m2 := map[int]int{
		1: 100,
		2: 200,
		3: 300,
		4: 400,
	}
	m3 := map[int]int{
		1: 1000,
		3: 3000,
		4: 4000,
	}
	combine := func(a, b int) int { return a + b }

	result := funk.IntersectMapsWith(combine, m1, m2, m3)
	expected := map[int]int{
		1: (10 + 100) + 1000,
		3: (30 + 300) + 3000,
		4: (40 + 400) + 4000,
	}
	assert.Equal(t, expected, result, "IntersectMapsWith should only include keys present in all maps and combine their values")
}

func TestIntersectMapsWithEmpty(t *testing.T) {
	m1 := map[int]int{
		1: 10,
		2: 20,
	}
	m2 := map[int]int{} // empty map
	combine := func(a, b int) int { return a + b }
	result := funk.IntersectMapsWith(combine, m1, m2)
	expected := map[int]int{}
	assert.Equal(t, expected, result, "IntersectMapsWith with an empty second map should return an empty map")
}

func TestIntersectMapsWithNil(t *testing.T) {
	var m1 map[int]int // nil map
	m2 := map[int]int{
		1: 100,
	}
	combine := func(a, b int) int { return a + b }
	result := funk.IntersectMapsWith(combine, m1, m2)
	expected := map[int]int{}
	assert.Equal(t, expected, result, "IntersectMapsWith with a nil first map should return an empty map")

	// Now test when the second map is nil.
	m1 = map[int]int{
		1: 10,
	}
	var m3 map[int]int // nil map as second argument
	result = funk.IntersectMapsWith(combine, m1, m3)
	assert.Equal(t, expected, result, "IntersectMapsWith with a nil second map should return an empty map")
}

func TestIntersectMapsWithStrings(t *testing.T) {
	m1 := map[string]string{
		"a": "hello",
		"b": "foo",
		"c": "bar",
	}
	m2 := map[string]string{
		"a": " world",
		"b": " baz",
		"d": " qux",
	}
	m3 := map[string]string{
		"a": "!!!",
		"b": "???",
		"c": "###", // key "c" is missing from m2, so it will be omitted.
	}
	combine := func(a, b string) string { return a + b }

	result := funk.IntersectMapsWith(combine, m1, m2, m3)
	expected := map[string]string{
		"a": "hello world!!!",
		"b": "foo baz???",
	}
	assert.Equal(t, expected, result, "IntersectMapsWith should correctly combine string values for common keys")
}

func TestDifferenceMaps_Basic(t *testing.T) {
	left := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	right := map[int]string{
		2: "two",
		4: "four",
	}

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{
		1: "one",
		3: "three",
	}
	expectedInRight := map[int]string{
		4: "four",
	}

	assert.Equal(t, expectedInLeft, inLeft, "Keys unique to left map are not as expected")
	assert.Equal(t, expectedInRight, inRight, "Keys unique to right map are not as expected")
}

func TestDifferenceMaps_Disjoint(t *testing.T) {
	left := map[int]string{
		1: "a",
		2: "b",
	}
	right := map[int]string{
		3: "c",
		4: "d",
	}

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{
		1: "a",
		2: "b",
	}
	expectedInRight := map[int]string{
		3: "c",
		4: "d",
	}

	assert.Equal(t, expectedInLeft, inLeft, "When maps are disjoint, left difference should equal the left map")
	assert.Equal(t, expectedInRight, inRight, "When maps are disjoint, right difference should equal the right map")
}

func TestDifferenceMaps_Identical(t *testing.T) {
	left := map[int]string{
		1: "a",
		2: "b",
	}
	right := map[int]string{
		1: "a",
		2: "b",
	}

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{}
	expectedInRight := map[int]string{}

	assert.Equal(t, expectedInLeft, inLeft, "When maps are identical, left difference should be empty")
	assert.Equal(t, expectedInRight, inRight, "When maps are identical, right difference should be empty")
}

func TestDifferenceMaps_LeftNil(t *testing.T) {
	var left map[int]string // nil map
	right := map[int]string{
		1: "one",
		2: "two",
	}

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{}
	expectedInRight := map[int]string{
		1: "one",
		2: "two",
	}

	assert.Equal(t, expectedInLeft, inLeft, "Nil left map should produce an empty left difference")
	assert.Equal(t, expectedInRight, inRight, "Right difference should equal right map when left is nil")
}

func TestDifferenceMaps_RightNil(t *testing.T) {
	left := map[int]string{
		1: "one",
		2: "two",
	}
	var right map[int]string // nil map

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{
		1: "one",
		2: "two",
	}
	expectedInRight := map[int]string{}

	assert.Equal(t, expectedInLeft, inLeft, "Left difference should equal left map when right is nil")
	assert.Equal(t, expectedInRight, inRight, "Nil right map should produce an empty right difference")
}

func TestDifferenceMaps_BothNil(t *testing.T) {
	var left map[int]string
	var right map[int]string

	inLeft, inRight := funk.DifferenceMaps(left, right)
	expectedInLeft := map[int]string{}
	expectedInRight := map[int]string{}

	assert.Equal(t, expectedInLeft, inLeft, "Both nil maps should produce an empty left difference")
	assert.Equal(t, expectedInRight, inRight, "Both nil maps should produce an empty right difference")
}

func TestKeysEmpty(t *testing.T) {
	m := map[int]string{}
	keys := funk.Keys(m)
	assert.Empty(t, keys, "An empty map should produce an empty keys slice")
}

func TestKeysNil(t *testing.T) {
	var m map[int]string
	keys := funk.Keys(m)
	assert.Empty(t, keys, "A nil map should produce an empty keys slice")
}

func TestKeysInt(t *testing.T) {
	m := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	keys := funk.Keys(m)
	expected := []int{1, 2, 3}
	assert.ElementsMatch(t, expected, keys, "Keys should contain all keys in the map regardless of order")
}

func TestKeysString(t *testing.T) {
	m := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	keys := funk.Keys(m)
	expected := []string{"a", "b", "c"}
	assert.ElementsMatch(t, expected, keys, "Keys should contain all keys in the map regardless of order")
}

func TestValuesEmpty(t *testing.T) {
	m := map[int]string{}
	values := funk.Values(m)
	assert.Empty(t, values, "An empty map should produce an empty values slice")
}

func TestValuesNil(t *testing.T) {
	var m map[int]string
	values := funk.Values(m)
	assert.Empty(t, values, "A nil map should produce an empty values slice")
}

func TestValuesString(t *testing.T) {
	m := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	values := funk.Values(m)
	expected := []string{"one", "two", "three"}
	assert.ElementsMatch(t, expected, values, "Values should contain all values in the map regardless of order")
}

func TestValuesInt(t *testing.T) {
	m := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	values := funk.Values(m)
	expected := []int{1, 2, 3}
	assert.ElementsMatch(t, expected, values, "Values should contain all values in the map regardless of order")
}

func TestMapFilter(t *testing.T) {
	m := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	expected := map[string]int{
		"b": 2,
		"c": 3,
	}

	actual := funk.MapFilter(m, func(key string, value int) bool {
		return value > 1
	})

	assert.Equal(t, expected, actual)
}

func TestMapKeysNil(t *testing.T) {
	var m1 map[int]string

	m2 := funk.MapKeys(m1, func(key int) float64 {
		panic("unexpected call")
	})

	assert.Nil(t, m1)
	assert.Equal(t, map[float64]string{}, m2, "mapping over an nil map should still produce an empty map")
}

func TestMapKeysEmpty(t *testing.T) {
	m1 := map[int]string{}

	m2 := funk.MapKeys(m1, func(key int) float64 {
		panic("unexpected call")
	})

	assert.Equal(t, map[int]string{}, m1, "mapping over keys shouldn't change the input map")
	assert.Equal(t, map[float64]string{}, m2, "mapping over an empty map should produce an empty map")
}

func TestMapKeys(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	m2 := funk.MapKeys(m1, func(key int) float64 {
		return float64(key * 2)
	})

	assert.Equal(t, map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}, m1, "mapping over keys shouldn't change the input map")

	assert.Equal(t, map[float64]string{
		2.0: "one",
		4.0: "two",
		6.0: "three",
	}, m2, "mapping over keys should produce the correct output map")
}

func TestMapKeysSameKey(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	m2 := funk.MapKeys(m1, func(key int) string {
		return "key"
	})

	assert.Equal(t, map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}, m1, "mapping over keys shouldn't change the input map")

	assert.Equal(t, map[string]string{
		"key": m2["key"],
	}, m2, "mapping over keys should produce the correct output map")
	assert.Contains(t, funk.Values(m1), m2["key"], "the value in the new map should be something from the old map")
}

func TestMapValuesNil(t *testing.T) {
	var m1 map[int]string

	m2 := funk.MapValues(m1, func(value string) float64 {
		panic("unexpected call")
	})

	assert.Nil(t, m1)
	assert.Equal(t, map[int]float64{}, m2, "mapping over an nil map should still produce an empty map")
}

func TestMapValuesEmpty(t *testing.T) {
	m1 := map[int]string{}

	m2 := funk.MapValues(m1, func(value string) float64 {
		panic("unexpected call")
	})

	assert.Equal(t, map[int]string{}, m1, "mapping over keys shouldn't change the input map")
	assert.Equal(t, map[int]float64{}, m2, "mapping over an empty map should produce an empty map")
}

func TestMapValues(t *testing.T) {
	m1 := map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}

	m2 := funk.MapValues(m1, func(value string) string {
		return value + " and more"
	})

	assert.Equal(t, map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}, m1, "mapping over values shouldn't change the input map")

	assert.Equal(t, map[int]string{
		1: "one and more",
		2: "two and more",
		3: "three and more",
	}, m2, "mapping over values should produce the correct output map")
}

func TestMapKeysWithEmpty(t *testing.T) {
	m := map[int]int{}
	result := funk.MapKeysWith(m, func(key int) int {
		return key * 2
	}, func(a, b int) int { return a + b })
	assert.Empty(t, result, "Empty map should produce an empty result")
}

func TestMapKeysWithNoConflict(t *testing.T) {
	original := map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}
	result := funk.MapKeysWith(original, func(key int) int {
		return key + 100
	}, func(a, b int) int { return a + b })

	expected := map[int]int{
		101: 10,
		102: 20,
		103: 30,
	}
	assert.Equal(t, expected, result, "Unique mapping should yield unchanged values")
	// Ensure original map is not modified.
	assert.Equal(t, map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}, original)
}

func TestMapKeysWithEvenOddConflict(t *testing.T) {
	original := map[int]int{
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}
	// Mapping function: odd keys become 1, even keys become 0.
	result := funk.MapKeysWith(original, func(key int) int {
		if key%2 == 0 {
			return 0
		}

		return 1
	}, func(a, b int) int { return a + b })

	expected := map[int]int{
		0: 2 + 4,
		1: 1 + 3,
	}
	assert.Equal(t, expected, result, "Mapping with even/odd conflict should combine values correctly")
	// Ensure original map is not modified.
	assert.Equal(t, map[int]int{
		1: 1,
		2: 2,
		3: 3,
		4: 4,
	}, original)
}

func TestMapKeysWithConstantKey(t *testing.T) {
	original := map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}
	// Mapping function returns a constant string key.
	result := funk.MapKeysWith(original, func(key int) string {
		return "all"
	}, func(a, b int) int { return a + b })

	expected := map[string]int{
		"all": 10 + 20 + 30,
	}
	assert.Equal(t, expected, result, "Mapping with constant key should combine all values")
	// Ensure original map is not modified.
	assert.Equal(t, map[int]int{
		1: 10,
		2: 20,
		3: 30,
	}, original)
}

func TestPopulateMapEmpty(t *testing.T) {
	result := funk.PopulateMap[string, int]("constant")
	expected := map[int]string{}
	assert.Equal(t, expected, result, "No keys provided should return an empty map")
}

func TestPopulateMapNonEmpty(t *testing.T) {
	result := funk.PopulateMap("constant", 1, 2, 3)
	expected := map[int]string{
		1: "constant",
		2: "constant",
		3: "constant",
	}
	assert.Equal(t, expected, result, "All provided keys should map to the given constant value")
}

func TestPopulateMapWithEmpty(t *testing.T) {
	result := funk.PopulateMapWith(func(key int) int { return key * 10 })
	expected := map[int]int{}
	assert.Equal(t, expected, result, "No keys provided should return an empty map")
}

func TestPopulateMapWithKeys(t *testing.T) {
	result := funk.PopulateMapWith(func(key string) int {
		return len(key)
	}, "apple", "banana", "cherry")
	expected := map[string]int{
		"apple":  5,
		"banana": 6,
		"cherry": 6,
	}
	assert.Equal(t, expected, result, "Generator should be applied to each provided key")
}

func TestPopulateMapWithGeneratorSideEffects(t *testing.T) {
	var callCount int
	generator := func(key int) int {
		callCount++

		return key * 2
	}

	result := funk.PopulateMapWith(generator, 1, 2, 3, 4)
	expected := map[int]int{
		1: 2,
		2: 4,
		3: 6,
		4: 8,
	}
	assert.Equal(t, expected, result, "Generator should correctly process each key")
	assert.Equal(t, 4, callCount, "Generator should be called once per key")
}
