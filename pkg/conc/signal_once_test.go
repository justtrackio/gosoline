package conc_test

import (
	"github.com/applike/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSignalOnce_Signal(t *testing.T) {
	cond := conc.NewSignalOnce()

	// should not initially signal
	select {
	case <-cond.Channel():
		assert.Fail(t, "unexpected read from channel")
	default:
		assert.False(t, cond.Signaled())
	}

	// after signal we should get a value
	cond.Signal()

	_, open := <-cond.Channel()
	assert.False(t, open)
	assert.True(t, cond.Signaled())

	// we should get another value
	_, open = <-cond.Channel()
	assert.False(t, open)
	assert.True(t, cond.Signaled())

	// we should be able to signal again
	cond.Signal()

	// and still get a value
	_, open = <-cond.Channel()
	assert.False(t, open)
	assert.True(t, cond.Signaled())
}
