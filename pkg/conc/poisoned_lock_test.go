package conc_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
)

func TestPoisonedLock(t *testing.T) {
	lck := conc.NewPoisonedLock()

	// should work
	lck.Lock()
	//nolint:golint,staticcheck
	lck.Unlock()

	// should also work
	{
		err := lck.TryLock()
		assert.NoError(t, err)
		lck.Unlock()
	}

	// now make sure we can't use it anymore
	lck.Poison()

	// should be okay to do again
	lck.Poison()

	{
		err := lck.TryLock()

		assert.Equal(t, conc.ErrAlreadyPoisoned, err)
	}

	var err error
	func() {
		defer func() {
			err = recover().(error)
		}()

		// should not be okay and set our error
		lck.Lock()
	}()

	assert.Equal(t, conc.ErrAlreadyPoisoned, err)

	// should still be okay and not cause our test to hang
	lck.Poison()
}
