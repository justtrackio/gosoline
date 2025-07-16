package conc_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/stretchr/testify/assert"
)

func TestLazyGet_Simple(t *testing.T) {
	var called int32
	l := conc.NewLazy(func(arg struct{}) (int, error) {
		atomic.AddInt32(&called, 1)

		return 1, nil
	})

	assert.Equal(t, int32(0), called)

	for i := 0; i < 10; i++ {
		result, err := l.Get(struct{}{})
		assert.Equal(t, result, 1)
		assert.NoError(t, err)
	}

	assert.Equal(t, int32(1), called)
}

func TestLazyGet_Error(t *testing.T) {
	var called int32
	l := conc.NewLazy(func(arg struct{}) (int, error) {
		atomic.AddInt32(&called, 1)

		return 0, fmt.Errorf("failed")
	})

	assert.Equal(t, int32(0), called)

	for i := 0; i < 10; i++ {
		_, err := l.Get(struct{}{})
		assert.EqualError(t, err, "failed")
		assert.Equal(t, int32(i+1), called)
	}
}

func TestLazyConcurrently(t *testing.T) {
	var called int32
	l := conc.NewLazy(func(arg int) (int, error) {
		atomic.AddInt32(&called, 1)

		if arg == 3 {
			return arg, nil
		}

		return 0, fmt.Errorf("arg was not 3")
	})

	ch := make(chan struct{})
	cfn := coffin.New(t.Context())
	for i := 0; i < 10; i++ {
		cfn.Go(fmt.Sprintf("runner %d", i), func() error {
			<-ch
			v, err := l.Get(i)
			if err != nil {
				assert.EqualError(t, err, "arg was not 3")
			} else {
				assert.Equal(t, 3, v)
			}

			return nil
		})
	}

	close(ch)

	err := cfn.Wait()
	assert.NoError(t, err)

	assert.LessOrEqual(t, called, int32(10))
}
