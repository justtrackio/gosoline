package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

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
	assert.Contains(t, maps.Values(m1), m2["key"], "the value in the new map should be something from the old map")
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
