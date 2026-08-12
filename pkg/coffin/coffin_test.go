package coffin_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/assert"
)

func TestCoffin_New(t *testing.T) {
	cfn := coffin.New()
	myErr := errors.New("my error")

	cfn.Gof(func() error {
		panic(myErr)
	}, "got this error: %d", 42)

	err := cfn.Wait()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, myErr))
	assert.True(t, strings.HasPrefix(err.Error(), "got this error: 42: my error"))
}

func TestCoffin_WithContext(t *testing.T) {
	// if you are asking wtf is this, you might be correct. But let me explain:
	// - we iterate a few times because this is a race condition and does not
	//   trigger every time
	// - the nested coffin pattern is actually used if your module is using the
	//   coffin module. The outer coffin is actually used by the kernel to keep
	//   track of your module
	// - the error we are testing for is "panic: close of closed channel"
	// - the error triggers because the old coffin implementation did COPY a coffin
	//   and by doing so did copy a mutex. but a mutex is not safe to copy after
	//   someone already got a reference to it - in this case tomb.WithContext
	// - thus, tomb.WithContext locked a DIFFERENT mutex than we later locked when
	//   killing the tomb/coffin, but closed the SAME channel
	// - this test is thus intended to make sure no one actually reintroduces this
	//   behavior
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("iteration %d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				runCoffinWithContextIteration(t)
			})
		})
	}
}

func runCoffinWithContextIteration(t *testing.T) {
	cfn, ctx := coffin.WithContext(t.Context())
	c := make(chan struct{})
	errStop := errors.New("please stop")

	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		nestedCfn, cfnCtx := coffin.WithContext(ctx)

		nestedCfn.GoWithContext(cfnCtx, func(ctx context.Context) error {
			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()
			count := 0

			for {
				select {
				case <-ticker.C:
					count++
					if count == 3 {
						close(c)
					}
				case <-ctx.Done():
					return nil
				}
			}
		})

		err := nestedCfn.Wait()
		if !errors.Is(err, context.Canceled) {
			assert.NoError(t, err)
		}

		return err
	})

	<-c
	cfn.Kill(errStop)
	err := cfn.Wait()

	assert.Equal(t, errStop, err)
}

func TestCoffin_WithContext_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cfn, ctx := coffin.WithContext(ctx)
	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		// if we exit this function before the coffin is dying, we might sometimes have it return nil, sometimes context.Canceled
		// thus, for now we wait until the coffin is dying, so we know it got already killed by context.Canceled
		<-cfn.Dying()

		return nil
	})

	cancel()

	err := cfn.Wait()
	assert.Equal(t, ctx.Err(), err)
}

func TestCoffin_Gof(t *testing.T) {
	cfn := coffin.New()
	cfn.Gof(func() error {
		var err error

		// crash the function!
		//goland:noinspection GoNilness
		errString := err.Error()
		assert.Failf(t, "got unexpected string back", errString)

		return err
	}, "crashing function")

	err := cfn.Wait()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "crashing function: runtime error: invalid memory address or nil pointer dereference")
}

func TestCoffin_Wait_Empty(t *testing.T) {
	cfn := coffin.New()
	// check waiting on an empty coffin does not block forever
	err := cfn.Wait()
	assert.NoError(t, err)
}

func TestCoffin_Wait_Goexit(t *testing.T) {
	// runtime.Goexit is what testify's t.FailNow (and therefore require.* and a failing mock expectation) triggers.
	// It skips tomb's alive bookkeeping, which used to make Wait block forever and turned any assertion failure
	// inside a tracked go routine into a hanging test instead of a failing one.
	cfn := coffin.New()
	cfn.Gof(func() error {
		runtime.Goexit()

		return nil
	}, "go routine calling Goexit")

	done := make(chan error, 1)
	go func() {
		done <- cfn.Wait()
	}()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, coffin.ErrGoexit)
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "Wait did not return after a tracked go routine called runtime.Goexit")
	}
}

func TestCoffin_Wait_GoexitWithSiblings(t *testing.T) {
	// a go routine lost to Goexit must still kill the coffin so its siblings get a chance to shut down
	cfn := coffin.New()
	sibling := make(chan struct{})

	cfn.Go(func() error {
		cfn.Gof(func() error {
			runtime.Goexit()

			return nil
		}, "go routine calling Goexit")

		cfn.Gof(func() error {
			<-cfn.Dying()
			close(sibling)

			return nil
		}, "sibling waiting for the coffin to die")

		return nil
	})

	done := make(chan error, 1)
	go func() {
		done <- cfn.Wait()
	}()

	select {
	case err := <-done:
		assert.ErrorIs(t, err, coffin.ErrGoexit)
		<-sibling
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "Wait did not return after a tracked go routine called runtime.Goexit")
	}
}
