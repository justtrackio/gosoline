package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
)

func TestSet_NewSet(t *testing.T) {
	s := funk.NewSet(1, 2, 3)

	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(2))
	assert.True(t, s.Contains(3))
	assert.Equal(t, 3, s.Len())
}

func TestSet_NewSetEmpty(t *testing.T) {
	s := funk.NewSet[int]()
	assert.True(t, s.Empty(), "NewSet with no arguments should be empty")
}

func TestSet_AddAndContains(t *testing.T) {
	s := make(funk.Set[int])

	assert.False(t, s.Contains(1), "set should not contain item 1 initially")

	s.Add(1)
	s.Add(2, 3)

	assert.True(t, s.Contains(1), "set should contain item 1")
	assert.True(t, s.Contains(2), "set should contain item 2")
	assert.True(t, s.Contains(3), "set should contain item 3")
	assert.False(t, s.Contains(4), "set should not contain item 4")
	assert.True(t, s.ContainsAll(funk.NewSet(1, 2, 3)), "set should contain 1, 2, 3")
	assert.Equal(t, 3, s.Len(), "set should contain 3 items")
}

func TestSet_AddDuplicates(t *testing.T) {
	s := make(funk.Set[int])
	s.Add(1)
	s.Add(1)
	s.Add(1)

	slice := s.ToSlice()
	expected := []int{1}
	assert.Equal(t, expected, slice, "Adding duplicates should not increase set size")
}

func TestSet_AddAll(t *testing.T) {
	a := funk.NewSet(1, 2)
	b := funk.NewSet(2, 3)

	changed := a.AddAll(b)
	assert.True(t, changed, "AddAll should return true when new items are added")

	expected := funk.NewSet(1, 2, 3)
	assert.True(t, a.ContainsAll(expected), "Set a should contain 1, 2, 3 after AddAll")
}

func TestSet_AddAll_NoChange(t *testing.T) {
	a := funk.NewSet(1, 2)
	b := funk.NewSet(1, 2)

	changed := a.AddAll(b)
	assert.False(t, changed, "AddAll should return false if no new items are added")
}

func TestSet_AddReturnValue(t *testing.T) {
	s := funk.NewSet[int]()
	assert.True(t, s.Add(1), "Add should return true when adding a new element")
	assert.False(t, s.Add(1), "Add should return false when adding a duplicate element")
}

func TestSet_AddAll_Empty(t *testing.T) {
	a := funk.NewSet(1, 2)
	b := funk.NewSet[int]() // empty

	changed := a.AddAll(b)
	assert.False(t, changed, "AddAll with an empty set should return false")
	assert.Equal(t, 2, a.Len(), "Original set should remain unchanged")
}

func TestSet_Remove(t *testing.T) {
	s := funk.NewSet(1, 2, 3)

	changed := s.Remove(2)
	assert.True(t, changed, "Remove should return true when item is removed")
	assert.False(t, s.Contains(2), "Set should no longer contain 2")
	assert.Equal(t, 2, s.Len())

	changed = s.Remove(42)
	assert.False(t, changed, "Remove should return false if item was not in the set")
}

func TestSet_RemoveEmptyArgs(t *testing.T) {
	s := funk.NewSet(1, 2)
	changed := s.Remove()
	assert.False(t, changed, "Remove with no arguments should return false")
	assert.Equal(t, 2, s.Len(), "Set should remain unchanged")
}

func TestSet_RemoveAll(t *testing.T) {
	a := funk.NewSet("a", "b", "c")
	b := funk.NewSet("b", "d")

	changed := a.RemoveAll(b)
	assert.True(t, changed, "RemoveAll should return true when items are removed")
	assert.False(t, a.Contains("b"), "Set should no longer contain 'b'")
	assert.True(t, a.Contains("a"))
	assert.True(t, a.Contains("c"))
}

func TestSet_RemoveAll_Empty(t *testing.T) {
	a := funk.NewSet(1, 2)
	b := funk.NewSet[int]() // empty

	changed := a.RemoveAll(b)
	assert.False(t, changed, "RemoveAll with an empty set should return false")
	assert.Equal(t, 2, a.Len(), "Original set should remain unchanged")
}

func TestSet_RemoveAll_NoChange(t *testing.T) {
	a := funk.NewSet("a", "b")
	b := funk.NewSet("x", "y")

	changed := a.RemoveAll(b)
	assert.False(t, changed, "RemoveAll should return false when no items are removed")
	assert.Equal(t, 2, a.Len())
}

func TestSet_ContainsAll(t *testing.T) {
	s := funk.NewSet(1, 2, 3)
	sub := funk.NewSet(1, 3)

	assert.True(t, s.ContainsAll(sub), "Set should contain all elements of the subset")

	sub.Add(42)
	assert.False(t, s.ContainsAll(sub), "Set should not contain all elements after adding 42")
}

func TestSet_Len(t *testing.T) {
	s := funk.NewSet[int]()
	assert.Equal(t, 0, s.Len())

	s.Add(5)
	assert.Equal(t, 1, s.Len())

	s.Add(5) // duplicate
	assert.Equal(t, 1, s.Len())
}

func TestSet_Empty(t *testing.T) {
	s := funk.NewSet[int]()
	assert.True(t, s.Empty(), "New set should be empty")

	s.Add(1)
	assert.False(t, s.Empty(), "Set should not be empty after adding an element")

	s.Remove(1)
	assert.True(t, s.Empty(), "Set should be empty after removing all elements")

	slice := s.ToSlice()
	assert.Empty(t, slice, "Empty set should yield an empty slice")
	assert.False(t, s.Contains(1), "Empty set should not contain any item")
}

func TestSet_ToSlice(t *testing.T) {
	s := make(funk.Set[string])
	s.Add("apple")
	s.Add("banana")
	s.Add("cherry")

	slice := funk.SetToSlice(s)
	expected := []string{"apple", "banana", "cherry"}
	assert.ElementsMatch(t, expected, slice, "SetToSlice should return all elements from the set")
}

func TestSet_ToSliceMethod(t *testing.T) {
	s := make(funk.Set[int])
	s.Add(10)
	s.Add(20)
	s.Add(30)

	slice := s.ToSlice()
	expected := []int{10, 20, 30}
	assert.ElementsMatch(t, expected, slice, "ToSlice should return all elements from the set")
}

func TestSet_SetToSlice_Empty(t *testing.T) {
	s := funk.NewSet[string]()
	slice := funk.SetToSlice(s)

	assert.Empty(t, slice, "SetToSlice should return empty slice for empty set")
}
