package funk

// A Set is a map which stores no values.
type Set[T comparable] map[T]struct{}

// NewSet creates a new Set with the given elements.
func NewSet[T comparable](items ...T) Set[T] {
	s := make(Set[T], len(items))
	s.Add(items...)

	return s
}

// Set adds a new value to the Set.
//
// Deprecated: Use Add instead
func (s Set[T]) Set(item T) {
	s[item] = struct{}{}
}

// Add adds new values to the Set.
//
// It returns true if any item was not contained in this Set.
func (s Set[T]) Add(items ...T) bool {
	result := false
	for _, item := range items {
		if _, ok := s[item]; ok {
			continue
		}

		s[item] = struct{}{}
		result = true
	}

	return result
}

// AddAll adds all elements of a Set to the Set.
//
// It returns true if any item was not contained in this Set.
func (s Set[T]) AddAll(items Set[T]) bool {
	result := false
	for item := range items {
		if s.Add(item) {
			result = true
		}
	}

	return result
}

// Remove removes values from the Set.
//
// It returns true if any item was contained in the Set.
func (s Set[T]) Remove(items ...T) bool {
	result := false
	for _, item := range items {
		if _, ok := s[item]; !ok {
			continue
		}

		delete(s, item)
		result = true
	}

	return result
}

// RemoveAll removes all elements of a Set from the Set.
//
// It returns true if any item was contained in the Set.
func (s Set[T]) RemoveAll(items Set[T]) bool {
	result := false
	for item := range items {
		if _, ok := s[item]; !ok {
			continue
		}

		delete(s, item)
		result = true
	}

	return result
}

// Contains checks if a Set contains the given value.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]

	return ok
}

// ContainsAll checks if a Set contains all values from the given other Set.
func (s Set[T]) ContainsAll(other Set[T]) bool {
	for item := range other {
		if !s.Contains(item) {
			return false
		}
	}

	return true
}

// Len returns the number of distinct elements in the Set.
func (s Set[T]) Len() int {
	return len(s)
}

// Empty returns true if the Set contains no entries.
func (s Set[T]) Empty() bool {
	return len(s) == 0
}

// SetToSlice converts a set to a slice containing the elements from the set.
// The order of the elements of the returned slice is undefined.
func SetToSlice[T comparable](s Set[T]) []T {
	out := make([]T, 0, len(s))
	for k := range s {
		out = append(out, k)
	}

	return out
}

// ToSlice is SetToSlice as a member function.
func (s Set[T]) ToSlice() []T {
	return SetToSlice(s)
}
