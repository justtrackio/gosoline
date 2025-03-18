package funk

// A Set is a map which stores no values.
type Set[T comparable] map[T]struct{}

// Set adds a new value to the set.
func (s Set[T]) Set(item T) {
	s[item] = struct{}{}
}

// Contains checks if a set contains the given value.
func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
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
