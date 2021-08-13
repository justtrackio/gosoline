package exec

import (
	"errors"
	"fmt"
	"time"
)

type ErrMaxElapsedTimeExceeded struct {
	Resource     *ExecutableResource
	Attempts     int
	DurationTook time.Duration
	DurationMax  time.Duration
	Err          error
}

func NewErrMaxElapsedTimeExceeded(resource *ExecutableResource, attempts int, durationTook time.Duration, durationMax time.Duration, err error) *ErrMaxElapsedTimeExceeded {
	return &ErrMaxElapsedTimeExceeded{
		Resource:     resource,
		Attempts:     attempts,
		DurationTook: durationTook,
		DurationMax:  durationMax,
		Err:          err,
	}
}

func (e ErrMaxElapsedTimeExceeded) Error() string {
	return fmt.Sprintf("sent request to resource %s failed after %d retries in %s: retry max duration %s exceeded: %s", e.Resource, e.Attempts, e.DurationTook, e.DurationMax, e.Err)
}

func (e ErrMaxElapsedTimeExceeded) Unwrap() error {
	return e.Err
}

func IsErrMaxElapsedTimeExceeded(err error) bool {
	var errExpected *ErrMaxElapsedTimeExceeded
	return errors.As(err, &errExpected)
}
