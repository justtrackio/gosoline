package coffin

import "sync/atomic"

type Void = struct{}

type Tomb interface {
	Coffin
	// Alive returns true if the Tomb is not in a dying or dead state.
	Alive() bool
	// Dead returns the channel that can be used to wait until all goroutines have finished running.
	Dead() <-chan Void
	// Dying returns the channel that can be used to wait until Coffin.Kill is called or the last go routine finished running.
	Dying() <-chan Void
}

type tomb struct {
	Coffin
	dead  <-chan Void
	dying <-chan Void
	alive *int32
}

func (c tomb) Alive() bool {
	return atomic.LoadInt32(c.alive) != 0
}

func (c tomb) Dead() <-chan Void {
	return c.dead
}

func (c tomb) Dying() <-chan Void {
	return c.dying
}
