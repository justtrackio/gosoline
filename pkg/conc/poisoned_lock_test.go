package conc_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
)

func TestPoisonedLock(t *testing.T) {
	lck := conc.NewPoisonedLock()

	// basic locking should work
	lck.MustLock()
	lck.Unlock()

	// should also work
	{
		err := lck.TryLock()
		assert.NoError(t, err)
		lck.Unlock()
	}

	// conditionally poisoning the lock and then not poisoning it should work
	{
		err := lck.PoisonIf(func() (bool, error) {
			return false, nil
		})
		assert.NoError(t, err)
	}

	// conditionally poisoning the lock and returning an error should work
	{
		err := lck.PoisonIf(func() (bool, error) {
			return false, fmt.Errorf("fail")
		})
		assert.Equal(t, fmt.Errorf("fail"), err)
	}

	// now make sure we can't use it anymore
	{
		err := lck.Poison()
		assert.NoError(t, err)
	}

	// should be okay to do again, this time it should report an error
	{
		err := lck.Poison()
		assert.Equal(t, conc.ErrAlreadyPoisoned, err)
	}

	// should be impossible to lock now
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
		lck.MustLock()
	}()

	assert.Equal(t, conc.ErrAlreadyPoisoned, err)

	// should still be okay and not cause our test to hang
	{
		err := lck.Poison()
		assert.Equal(t, conc.ErrAlreadyPoisoned, err)
	}
}

func TestPoisonedLock_PoisonIfNoError(t *testing.T) {
	lck := conc.NewPoisonedLock()

	{
		err := lck.PoisonIf(func() (bool, error) {
			return true, nil
		})
		assert.NoError(t, err)
	}

	{
		err := lck.TryLock()
		assert.Equal(t, conc.ErrAlreadyPoisoned, err)
	}
}

func TestPoisonedLock_PoisonIfWithError(t *testing.T) {
	lck := conc.NewPoisonedLock()

	{
		err := lck.PoisonIf(func() (bool, error) {
			return true, fmt.Errorf("fail")
		})
		assert.Equal(t, fmt.Errorf("fail"), err)
	}

	{
		err := lck.TryLock()
		assert.Equal(t, conc.ErrAlreadyPoisoned, err)
	}
}
