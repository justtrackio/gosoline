package refl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/stretchr/testify/assert"
)

func TestSliceInterfaceIterator(t *testing.T) {
	slice := []int{0, 1, 2, 3, 4}
	actual := make([]int, 0)

	it := refl.SliceInterfaceIterator(slice)

	for it.Next() {
		val := it.Val()
		actual = append(actual, val.(int))
	}

	assert.Equal(t, slice, actual)
}

func TestSliceOf(t *testing.T) {
	s0 := make([]int, 0)
	_, err := refl.SliceOf(s0)
	assert.EqualError(t, err, "the slice has to be addressable", "it should fail if the slice is not addressable")

	s1 := make([]int, 0)
	rs, err := refl.SliceOf(&s1)
	assert.NoError(t, err)

	i1 := 4
	err = rs.Append(i1)
	assert.NoError(t, err)

	i2 := 5
	err = rs.Append(i2)
	assert.NoError(t, err)

	assert.Len(t, s1, 2)
	assert.Equal(t, 4, s1[0])
	assert.Equal(t, 5, s1[1])

	s2 := make([]*int, 0)
	rs, err = refl.SliceOf(&s2)
	assert.NoError(t, err)

	i0 := 0
	err = rs.Append(i0)
	assert.EqualError(t, err, "the value which you try to append to the slice has to be addressable")

	i1 = 1
	err = rs.Append(&i1)
	assert.NoError(t, err)

	i2 = 2
	err = rs.Append(&i2)
	assert.NoError(t, err)

	assert.Len(t, s2, 2)
	assert.Equal(t, 1, *s2[0])
	assert.Equal(t, 2, *s2[1])
}

func TestFlatten(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := refl.Flatten()
		assert.Empty(t, result)
	})

	t.Run("single non-slice element", func(t *testing.T) {
		result := refl.Flatten(42)
		assert.Equal(t, []any{42}, result)
	})

	t.Run("multiple non-slice elements", func(t *testing.T) {
		result := refl.Flatten(1, "hello", 3.14, true)
		assert.Equal(t, []any{1, "hello", 3.14, true}, result)
	})

	t.Run("single slice", func(t *testing.T) {
		result := refl.Flatten([]int{1, 2, 3})
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("single array", func(t *testing.T) {
		arr := [3]int{1, 2, 3}
		result := refl.Flatten(arr)
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("multiple slices", func(t *testing.T) {
		result := refl.Flatten([]int{1, 2}, []int{3, 4})
		assert.Equal(t, []any{1, 2, 3, 4}, result)
	})

	t.Run("mixed slices and non-slices", func(t *testing.T) {
		result := refl.Flatten(1, []int{2, 3}, 4, []string{"a", "b"})
		assert.Equal(t, []any{1, 2, 3, 4, "a", "b"}, result)
	})

	t.Run("nested slices", func(t *testing.T) {
		result := refl.Flatten([]any{1, []int{2, 3}, 4})
		// Note: This only flattens one level
		assert.Equal(t, []any{1, []int{2, 3}, 4}, result)
	})

	t.Run("empty slices", func(t *testing.T) {
		result := refl.Flatten([]int{}, 1, []string{}, 2)
		assert.Equal(t, []any{1, 2}, result)
	})

	t.Run("slice of different types", func(t *testing.T) {
		result := refl.Flatten([]int{1, 2}, []string{"a", "b"}, []float64{3.14, 2.71})
		assert.Equal(t, []any{1, 2, "a", "b", 3.14, 2.71}, result)
	})

	t.Run("slice with nil elements", func(t *testing.T) {
		result := refl.Flatten([]any{1, nil, 3})
		assert.Equal(t, []any{1, nil, 3}, result)
	})

	t.Run("nil as element", func(t *testing.T) {
		result := refl.Flatten(1, nil, 3)
		assert.Equal(t, []any{1, nil, 3}, result)
	})

	t.Run("slice of pointers", func(t *testing.T) {
		a, b, c := 1, 2, 3
		result := refl.Flatten([]*int{&a, &b, &c})
		assert.Len(t, result, 3)
		assert.Equal(t, 1, *result[0].(*int))
		assert.Equal(t, 2, *result[1].(*int))
		assert.Equal(t, 3, *result[2].(*int))
	})

	t.Run("pointer to slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		result := refl.Flatten(&slice)
		// Pointers to slices are dereferenced and flattened
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("pointer to array", func(t *testing.T) {
		arr := [3]int{1, 2, 3}
		result := refl.Flatten(&arr)
		// Pointers to arrays are dereferenced and flattened
		assert.Equal(t, []any{1, 2, 3}, result)
	})

	t.Run("pointer elements mixed with values", func(t *testing.T) {
		a := 42
		result := refl.Flatten(1, &a, 3)
		assert.Len(t, result, 3)
		assert.Equal(t, 1, result[0])
		assert.Equal(t, &a, result[1])
		assert.Equal(t, 3, result[2])
	})
}
