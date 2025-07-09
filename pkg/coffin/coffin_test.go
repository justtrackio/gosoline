package coffin_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/assert"
)

func TestCoffin_New(t *testing.T) {
	cfn := coffin.NewGraveyard().Entomb()
	myErr := errors.New("my error")

	cfn.Go("test", func() error {
		panic(myErr)
	}, coffin.WithErrorWrapper("got this error: %d", 42))

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
				cfn := coffin.NewGraveyard(coffin.WithContext(context.Background())).Entomb()
				c := make(chan struct{})
				errStop := errors.New("please stop")

				cfn.GoWithContext("test", func(ctx context.Context) error {
					nestedCfn := coffin.NewGraveyard(coffin.WithContext(ctx)).Entomb()

					nestedCfn.GoWithContext("inner test", func(ctx context.Context) error {
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

					return nil
				})

				<-c
				cfn.Kill(errStop)
				err := cfn.Wait()

				assert.EqualError(t, err, errStop.Error())
			})
		})
	}
}

func TestCoffin_WithContext_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfn := coffin.NewGraveyard(coffin.WithContext(ctx)).Entomb()
	cfn.GoWithContext("test", func(ctx context.Context) error {
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
	cfn := coffin.NewGraveyard().Entomb()
	cfn.Go("test", func() error {
		var err error

		// crash the function!
		//goland:noinspection GoNilness
		errString := err.Error()
		assert.Failf(t, "got unexpected string back", errString)

		return err
	}, coffin.WithErrorWrapper("crashing function"))

	err := cfn.Wait()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "crashing function: runtime error: invalid memory address or nil pointer dereference")
}

func TestCoffin_Wait_Empty(t *testing.T) {
	cfn := coffin.NewGraveyard().Entomb()
	// check waiting on an empty coffin does not block forever
	err := cfn.Wait()
	assert.NoError(t, err)
}
