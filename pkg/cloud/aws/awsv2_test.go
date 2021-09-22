package aws_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/stretchr/testify/assert"
)

func TestExponentialBackoffDelayer(t *testing.T) {
	initialInterval := time.Millisecond * 100
	maxInerval := time.Minute

	delayer := aws.NewBackoffDelayer(initialInterval, maxInerval)

	i := 1
	last := time.Duration(0)

	for ; i <= 100; i++ {
		delay, _ := delayer.BackoffDelay(i, nil)
		fmt.Printf("%02d: %s\n", i, delay)

		assert.True(t, delay > 0)
		assert.True(t, delay <= maxInerval)

		if delay == maxInerval && delay == last {
			break
		}

		last = delay
	}

	assert.True(t, i < 100)
}
