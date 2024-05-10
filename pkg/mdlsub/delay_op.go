package mdlsub

import (
	"errors"
	"fmt"
)

type DelayOpError struct {
	Err error
}

func (e DelayOpError) Unwrap() error {
	return e.Err
}

func NewDelayOpError(err error) DelayOpError {
	return DelayOpError{
		Err: err,
	}
}

func IsDelayOpError(err error) bool {
	return errors.As(err, &DelayOpError{})
}

func (e DelayOpError) Error() string {
	return fmt.Sprintf("delayed op error: %s", e.Err.Error())
}

func (e DelayOpError) As(target any) bool {
	if t, ok := target.(*DelayOpError); ok {
		*t = e

		return true
	}

	return false
}
