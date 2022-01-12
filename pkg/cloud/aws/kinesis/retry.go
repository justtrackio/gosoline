package kinesis

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
)

type BackoffDelayer struct {
	*gosoAws.BackoffDelayer
	readProvisionedThroughputDelay time.Duration
}

func NewBackoffDelayer(initialInterval time.Duration, maxInterval time.Duration, readProvisionedThroughputDelay time.Duration) *BackoffDelayer {
	return &BackoffDelayer{
		BackoffDelayer:                 gosoAws.NewBackoffDelayer(initialInterval, maxInterval),
		readProvisionedThroughputDelay: readProvisionedThroughputDelay,
	}
}

func (d *BackoffDelayer) BackoffDelay(attempt int, err error) (time.Duration, error) {
	var pte *types.ProvisionedThroughputExceededException
	if errors.As(err, &pte) {
		return d.readProvisionedThroughputDelay, nil
	}

	return d.BackoffDelayer.BackoffDelay(attempt, err)
}
