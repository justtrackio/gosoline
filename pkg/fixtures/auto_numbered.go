package fixtures

import "github.com/justtrackio/gosoline/pkg/mdl"

type AutoNumbered struct {
	nextId uint
}

func NewAutoNumbered() *AutoNumbered {
	return &AutoNumbered{
		nextId: 1,
	}
}

func NewAutoNumberedFrom(initial uint) *AutoNumbered {
	return &AutoNumbered{
		nextId: initial,
	}
}

// GetNext provides a fresh id for a fixture in case you don't want to assign specific ids
// Keep in mind that the ids are unique only in the scope of the same *AutoNumbered instance
func (n *AutoNumbered) GetNext() *uint {
	result := mdl.Box(n.nextId)
	n.nextId++

	return result
}
