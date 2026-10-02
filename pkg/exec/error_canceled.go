package exec

import (
	"context"
	"errors"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/funk"
)

const RequestCanceledError = requestCanceledError("RequestCanceled")

type requestCanceledError string

func (e requestCanceledError) Error() string {
	return string(e)
}

func CheckRequestCanceled(_ any, err error) ErrorType {
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

// IsRequestCanceled checks if the given error was caused by a canceled context - even if there is any other error contained in it, we
// return true. Thus, if IsRequestCanceled returns true, you can expect the context got canceled somewhere during that operation.
func IsRequestCanceled(err error) bool {
	type multipleErrors interface {
		Unwrap() []error
	}

	if multiErr, ok := err.(multipleErrors); ok {
		// check if one of the errors is request canceled
		if funk.Any(multiErr.Unwrap(), IsRequestCanceled) {
			return true
		}
	}

	multiErr := &multierror.Error{}
	if errors.As(err, &multiErr) {
		// check if one of the errors is request canceled
		if funk.Any(multiErr.Errors, IsRequestCanceled) {
			return true
		}
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
