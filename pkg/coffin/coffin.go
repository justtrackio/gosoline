package coffin

import "sync/atomic"

type Void = struct{}

type Coffin interface {
	Graveyard
	// Alive returns true if the Coffin is not in a dying or dead state.
	Alive() bool
	// Dead returns the channel that can be used to wait until all goroutines have finished running.
	Dead() <-chan Void
	// Dying returns the channel that can be used to wait until Kill is called.
	Dying() <-chan Void
	// Kill puts the Coffin in a dying state for the given reason, closes the Dying channel, and sets Alive to false.
	//
	// Although Kill may be called multiple times, only the first non-nil error is recorded as the death reason.
	Kill(reason error)
}

type coffin struct {
	Graveyard
	kill  func(reason error)
	dead  <-chan Void
	dying <-chan Void
	alive *int32
}

// New creates a new Coffin.
//
// Deprecated: use NewGraveyard instead, most of the time you only need a Graveyard and no Coffin
func New() Coffin {
	return NewGraveyard().Entomb()
}

func (c coffin) Alive() bool {
	return atomic.LoadInt32(c.alive) != 0
}

func (c coffin) Dead() <-chan Void {
	return c.dead
}

func (c coffin) Dying() <-chan Void {
	return c.dying
}

func (c coffin) Kill(reason error) {
	c.kill(reason)
}
