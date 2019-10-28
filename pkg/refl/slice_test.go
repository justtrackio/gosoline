package refl_test

import (
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSliceOf(t *testing.T) {
	s0 := make([]int, 0)
	_, err := refl.SliceOf(s0)
	assert.EqualError(t, err, "the slice has to be addressable", "it should fail if the slice is not addressable")

	s1 := make([]int, 0)
	rs, _ := refl.SliceOf(&s1)

	i1 := rs.NewElement()
	i1 = 4
	_ = rs.Append(i1)

	i2 := rs.NewElement()
	i2 = 5
	_ = rs.Append(i2)

	assert.Len(t, s1, 2)
	assert.Equal(t, 4, s1[0])
	assert.Equal(t, 5, s1[1])

	s2 := make([]*int, 0)
	rs, _ = refl.SliceOf(&s2)

	i0 := rs.NewElement()
	i0 = 0
	err = rs.Append(i0)

	i1 = rs.NewElement()
	i1 = mdl.Int(1)
	_ = rs.Append(i1)

	i2 = rs.NewElement()
	i2 = mdl.Int(2)
	_ = rs.Append(i2)

	assert.EqualError(t, err, "the value which you try to append to the slice has to be addressable", "the element has to be addressable")
	assert.Len(t, s2, 2)
	assert.Equal(t, 1, *s2[0])
	assert.Equal(t, 2, *s2[1])
}
