package fixtures

// AutoNumbered is the old name for a NumberSequence
//
// Deprecated: use NumberSequence instead
type AutoNumbered NumberSequence

// NewAutoNumbered is the old name for NewNumberSequence
//
// Deprecated: use NewNumberSequence instead
func NewAutoNumbered() AutoNumbered {
	return NewNumberSequence()
}

// NewAutoNumberedFrom is the old name for NewNumberSequenceFrom
//
// Deprecated: use NewNumberSequenceFrom instead
func NewAutoNumberedFrom(initial uint) AutoNumbered {
	return NewNumberSequenceFrom(initial)
}
