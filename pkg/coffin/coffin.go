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
}

type coffin struct {
	Graveyard
	dead  <-chan Void
	dying <-chan Void
	alive *int32
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
