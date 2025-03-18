package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
)

func TestSet_AddAndContains(t *testing.T) {
	s := make(funk.Set[int])

	assert.False(t, s.Contains(1), "set should not contain item 1 initially")

	s.Set(1)
	s.Set(2)
	s.Set(3)

	assert.True(t, s.Contains(1), "set should contain item 1")
	assert.True(t, s.Contains(2), "set should contain item 2")
	assert.True(t, s.Contains(3), "set should contain item 3")
	assert.False(t, s.Contains(4), "set should not contain item 4")
}

func TestSet_ToSlice(t *testing.T) {
	s := make(funk.Set[string])
	s.Set("apple")
	s.Set("banana")
	s.Set("cherry")

	slice := funk.SetToSlice(s)
	expected := []string{"apple", "banana", "cherry"}
	assert.ElementsMatch(t, expected, slice, "SetToSlice should return all elements from the set")
}

func TestSet_ToSliceMethod(t *testing.T) {
	s := make(funk.Set[int])
	s.Set(10)
	s.Set(20)
	s.Set(30)

	slice := s.ToSlice()
	expected := []int{10, 20, 30}
	assert.ElementsMatch(t, expected, slice, "ToSlice should return all elements from the set")
}

func TestSet_AddDuplicates(t *testing.T) {
	s := make(funk.Set[int])
	s.Set(1)
	s.Set(1)
	s.Set(1)

	slice := s.ToSlice()
	expected := []int{1}
	assert.Equal(t, expected, slice, "Adding duplicates should not increase set size")
}

func TestSet_Empty(t *testing.T) {
	s := make(funk.Set[string])
	slice := s.ToSlice()
	assert.Empty(t, slice, "Empty set should yield an empty slice")
	assert.False(t, s.Contains("anything"), "Empty set should not contain any item")
}
