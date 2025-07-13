package exec

import (
	"context"
	"errors"

	multierror "github.com/hashicorp/go-multierror"
)

const RequestCanceledError = requestCanceledError("RequestCanceled")

type requestCanceledError string

func (e requestCanceledError) Error() string {
	return string(e)
}

func CheckRequestCanceled(_ interface{}, err error) ErrorType {
	if IsRequestCanceled(err) {
		return ErrorTypePermanent
	}

	return ErrorTypeUnknown
}

type RequestCanceledCheck func(err error) bool

var requestCancelChecks = []RequestCanceledCheck{
	isError(context.Canceled),
	isError(context.DeadlineExceeded),
	isError(RequestCanceledError),
}

func AddRequestCancelCheck(check RequestCanceledCheck) {
	requestCancelChecks = append(requestCancelChecks, check)
}

// IsRequestCanceled checks if the given error was (only) caused by a canceled context - if there is any other error contained in it, we
// return false. Thus, if IsRequestCanceled returns true, you can (and should) ignore the error and stop processing instead.
func IsRequestCanceled(err error) bool {
	type multipleErrors interface {
		Unwrap() []error
	}

	if multiErr, ok := err.(multipleErrors); ok {
		// check if one of the errors is no request canceled
		for _, err := range multiErr.Unwrap() {
			if !IsRequestCanceled(err) {
				return false
			}
		}

		// all errors are a canceled request (if there are any)
		return len(multiErr.Unwrap()) > 0
	}

	multiErr := &multierror.Error{}
	if errors.As(err, &multiErr) {
		// check if one of the errors is no request canceled
		for _, err := range multiErr.Errors {
			if !IsRequestCanceled(err) {
				return false
			}
		}

		return len(multiErr.Errors) > 0
	}

	for _, check := range requestCancelChecks {
		if check(err) {
			return true
		}
	}

	return false
}

func isError(target error) RequestCanceledCheck {
	return func(err error) bool {
		return errors.Is(err, target)
	}
}
