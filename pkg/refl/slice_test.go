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
	_ = rs.Append(i1)

	i2 := 5
	_ = rs.Append(i2)

	assert.Len(t, s1, 2)
	assert.Equal(t, 4, s1[0])
	assert.Equal(t, 5, s1[1])

	s2 := make([]*int, 0)
	rs, _ = refl.SliceOf(&s2)

	i0 := 0
	err = rs.Append(i0)

	i1 = 1
	_ = rs.Append(&i1)

	i2 = 2
	_ = rs.Append(&i2)

	assert.EqualError(t, err, "the value which you try to append to the slice has to be addressable", "the element has to be addressable")
	assert.Len(t, s2, 2)
	assert.Equal(t, 1, *s2[0])
	assert.Equal(t, 2, *s2[1])
}
