package funk

type Set[T comparable] map[T]struct{}

func (s Set[T]) Set(item T) {
	s[item] = struct{}{}
}

func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}
