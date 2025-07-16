package aws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/assert"
)

func TestBackoffDelayer_Simple(t *testing.T) {
	delayer := aws.NewBackoffDelayer(time.Second, time.Second*10)

	for i := 0; i < 100; i++ {
		delay, err := delayer.BackoffDelay(i, nil)

		assert.NoError(t, err)
		assert.GreaterOrEqual(t, delay, time.Duration(0))
		assert.LessOrEqual(t, delay, time.Second*10)
	}
}

func TestBackoffDelayer_Concurrent(t *testing.T) {
	delayer := aws.NewBackoffDelayer(time.Second, time.Second*10)

	cfn := coffin.New(t.Context())
	timeout := make(chan struct{})

	for i := 0; i < 100; i++ {
		cfn.GoWithContext(fmt.Sprintf("task %d", i), func(ctx context.Context) error {
			delays := make([]time.Duration, 0)
			defer func() {
				for _, delay := range delays {
					assert.GreaterOrEqual(t, delay, time.Duration(0))
					assert.LessOrEqual(t, delay, time.Second*10)
				}
			}()

			for {
				select {
				case <-timeout:
					return nil
				default:
					for j := 0; j < 100; j++ {
						delay, err := delayer.BackoffDelay(j, nil)
						// we must not call any asserts here! if we do so, we lock a mutex on t, causing us to
						// synchronize this code which should expose possible races. So instead do a cheap error
						// check and add the delay to a local list to be checked later
						if err != nil {
							return err
						}

						delays = append(delays, delay)
					}
				}
			}
		})
	}

	cfn.Go("timeout task", func() error {
		time.Sleep(time.Second)
		close(timeout)

		return nil
	})

	assert.NoError(t, cfn.Wait())
}
