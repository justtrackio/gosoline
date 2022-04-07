package funk

type Set[T comparable] map[T]struct{}

func (s Set[T]) Set(item T) {
	s[item] = struct{}{}
}

func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

func SetToSlice[T comparable](s Set[T]) []T {
	out := make([]T, 0, len(s))
	for k := range s {
		out = append(out, k)
	}

	return out
}
