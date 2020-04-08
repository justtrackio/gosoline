package cloud

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/hashicorp/go-multierror"
	"io"
	"net"
	"net/url"
	"os"
	"syscall"
)

func isAwsError(err error, awsCode string) bool {
	var aerr awserr.Error
	if errors.As(err, &aerr) {
		return aerr.Code() == awsCode
	}

	return false
}

// Check if the given error was (only) caused by a canceled context - if there is any other error contained in it, we
// return false. Thus, if IsRequestCanceled returns true, you can (and should) ignore the error and stop processing instead.
func IsRequestCanceled(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, RequestCanceledError) || isAwsError(err, request.CanceledErrorCode) {
		return true
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

func IsUsedClosedConnectionError(err error) bool {
	opErr, ok := opError(err)

	if !ok {
		return false
	}

	return opErr.Err.Error() == "use of closed network connection"
}

func IsConnectionError(err error) bool {
	if isUrlError(err, io.EOF) {
		return true
	}

	if isSyscallError(err, syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.EPIPE) {
		return true
	}

	return false
}

func isUrlError(err error, targets ...error) bool {
	urlErr, ok := urlError(err)

	if !ok {
		return false
	}

	for _, t := range targets {
		if errors.Is(urlErr, t) {
			return true
		}
	}

	return false
}

func isSyscallError(err error, syscallErrors ...syscall.Errno) bool {
	opErr, ok := opError(err)

	if !ok {
		return false
	}

	for {
		if nextOpErr, ok := opErr.Err.(*net.OpError); ok {
			opErr = nextOpErr
		} else {
			break
		}
	}

	syscallErr, ok := opErr.Err.(*os.SyscallError)

	if !ok {
		return false
	}

	for _, sysErr := range syscallErrors {
		if syscallErr.Err == sysErr {
			return true
		}
	}

	return false
}

func urlError(err error) (*url.Error, bool) {
	if err == nil {
		return nil, false
	}

	aerr, ok := err.(awserr.Error)

	if !ok {
		return nil, false
	}

	urlErr, ok := aerr.OrigErr().(*url.Error)

	return urlErr, ok
}

func opError(err error) (*net.OpError, bool) {
	urlErr, ok := urlError(err)

	if !ok {
		return nil, false
	}

	opErr, ok := urlErr.Err.(*net.OpError)

	return opErr, ok
}

const RequestCanceledError = requestCanceledError("RequestCanceled")

type requestCanceledError string

func (e requestCanceledError) Error() string {
	return string(e)
}
