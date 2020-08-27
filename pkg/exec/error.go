package exec

import (
	"context"
	"errors"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sys/unix"
	"io"
	"strings"
)

type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorOk
	ErrorPermanent
	ErrorRetryable
)

const RequestCanceledError = requestCanceledError("RequestCanceled")

type requestCanceledError string

func (e requestCanceledError) Error() string {
	return string(e)
}

type ErrorChecker func(result interface{}, err error) ErrorType

func CheckRequestCanceled(_ interface{}, err error) ErrorType {
	if IsRequestCanceled(err) {
		return ErrorPermanent
	}

	return ErrorUnknown
}

type RequestCanceledChek func(err error) bool

var requestCancelChecks = make([]RequestCanceledChek, 0)

func AddRequestCancelCheck(check RequestCanceledChek) {
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

func CheckUsedClosedConnectionError(_ interface{}, err error) ErrorType {
	if IsUsedClosedConnectionError(err) {
		return ErrorRetryable
	}

	return ErrorUnknown
}

func IsUsedClosedConnectionError(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}

func CheckConnectionError(_ interface{}, err error) ErrorType {
	if IsConnectionError(err) {
		return ErrorRetryable
	}

	return ErrorUnknown
}

func IsConnectionError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, unix.ECONNREFUSED) || errors.Is(err, unix.ECONNRESET) || errors.Is(err, unix.EPIPE) {
		return true
	}

	return false
}
