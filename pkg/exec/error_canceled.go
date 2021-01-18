package exec

import (
	"context"
	"errors"
	"github.com/hashicorp/go-multierror"
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

var requestCancelChecks = make([]RequestCanceledCheck, 0)

func AddRequestCancelCheck(check RequestCanceledCheck) {
	requestCancelChecks = append(requestCancelChecks, check)
}

// Check if the given error was (only) caused by a canceled context - if there is any other error contained in it, we
// return false. Thus, if IsRequestCanceled returns true, you can (and should) ignore the error and stop processing instead.
func IsRequestCanceled(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, RequestCanceledError) {
		return true
	}

	for _, check := range requestCancelChecks {
		if check(err) {
			return true
		}
	}

	// some functions (like a batched Write) return a multierror which does not properly unwrap
	// so we check if all these requests failed. If there is any other error, we return false to
	// trigger normal error handling
	var multiErr *multierror.Error
	if errors.As(err, &multiErr) && multiErr != nil {
		for _, err := range multiErr.Errors {
			if !IsRequestCanceled(err) {
				return false
			}
		}

		return len(multiErr.Errors) > 0
	}

	return false
}
