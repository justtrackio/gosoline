package fixtures

import (
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

type NumberSequence interface {
	// GetNextInt returns the next number in the sequence as an integer
	GetNextInt() int
	// GetNextUint returns the next number in the sequence as an unsigned integer
	GetNextUint() uint
	// GetNextId is the same as GetNextUint, but returns a pointer to the result.
	// The returned id can be used to get a fresh id for a fixture if you don't want to
	// assign specific ids.
	GetNextId() *uint
	// GetNext is the old name of GetNextId
	// Deprecated: use GetNextId instead
	GetNext() *uint
}

type numberSequence struct {
	nextId uint64
}

func NewNumberSequence() NumberSequence {
	return NewNumberSequenceFrom(1)
}

func NewNumberSequenceFrom(initial uint) NumberSequence {
	return &numberSequence{
		nextId: uint64(initial),
	}
}

func (n *numberSequence) GetNextInt() int {
	return int(n.GetNextUint())
}

func (n *numberSequence) GetNextUint() uint {
	return uint(atomic.AddUint64(&n.nextId, 1) - 1)
}

func (n *numberSequence) GetNextId() *uint {
	return mdl.Box(n.GetNextUint())
}

func (n *numberSequence) GetNext() *uint {
	return n.GetNextId()
}
