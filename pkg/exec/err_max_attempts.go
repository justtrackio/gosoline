package exec

import (
	"errors"
	"fmt"
	"time"
)

type ErrAttemptsExceeded struct {
	Resource     *ExecutableResource
	Attempts     int
	DurationTook time.Duration
	Err          error
}

func NewErrAttemptsExceeded(resource *ExecutableResource, attempts int, durationTook time.Duration, err error) *ErrAttemptsExceeded {
	return &ErrAttemptsExceeded{
		Resource:     resource,
		Attempts:     attempts,
		DurationTook: durationTook,
		Err:          err,
	}
}

func (e *ErrAttemptsExceeded) Error() string {
	return fmt.Sprintf("sent request to resource %s failed after exceeding max attempts of %d retries in %s: %s", e.Resource, e.Attempts, e.DurationTook, e.Err)
}

func (e *ErrAttemptsExceeded) Unwrap() error {
	return e.Err
}

func IsErrMaxAttemptsExceeded(err error) bool {
	var errExpected *ErrAttemptsExceeded

	return errors.As(err, &errExpected)
}
