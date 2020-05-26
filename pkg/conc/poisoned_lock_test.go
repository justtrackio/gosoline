package conc_test

import (
	"github.com/applike/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPoisonedLock(t *testing.T) {
	lck := conc.NewPoisonedLock()

	// should work
	lck.Lock()
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

		assert.Equal(t, conc.AlreadyPoisonedErr, err)
	}

	var err error
	func() {
		defer func() {
			err = recover().(error)
		}()

		// should not be okay and set our error
		lck.Lock()
	}()

	assert.Equal(t, conc.AlreadyPoisonedErr, err)

	// should still be okay and not cause our test to hang
	lck.Poison()
}
