package conc_test

import (
	"sync/atomic"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
)

func TestAtomicBoolean(t *testing.T) {
	var b conc.AtomicBoolean
	var tCount int32
	var fCount int32

	assert.False(t, b.Get())
	b.Set(true)
	assert.True(t, b.Get())
	b.Set(false)
	assert.False(t, b.Get())
	cfn := coffin.New(func(cfn coffin.StartingCoffin) {
		for i := 0; i < 10; i++ {
			cfn.Go(func() error {
				for j := 0; j < 1000; j++ {
					if b.Flip() {
						atomic.AddInt32(&tCount, 1)
					} else {
						atomic.AddInt32(&fCount, 1)
					}
				}

				return nil
			})
		}
	})

	err := cfn.Wait()
	assert.NoError(t, err)

	assert.False(t, b.Get())
	assert.Equal(t, int32(5000), tCount)
	assert.Equal(t, int32(5000), fCount)
}
