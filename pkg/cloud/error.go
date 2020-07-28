package cloud

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/sys/unix"
	"io"
	"strings"
)

func IsAwsError(err error, awsCode string) bool {
	var aerr awserr.Error
	if errors.As(err, &aerr) {
		return aerr.Code() == awsCode
	}

	return false
}

// Check if the given error was (only) caused by a canceled context - if there is any other error contained in it, we
// return false. Thus, if IsRequestCanceled returns true, you can (and should) ignore the error and stop processing instead.
func IsRequestCanceled(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, RequestCanceledError) || IsAwsError(err, request.CanceledErrorCode) {
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
	return strings.Contains(err.Error(), "use of closed network connection")
}

func IsConnectionError(err error) bool {
	var aerr awserr.Error
	if errors.As(err, &aerr) {
		err = aerr.OrigErr()
	}

	if errors.Is(err, io.EOF) || errors.Is(err, unix.ECONNREFUSED) || errors.Is(err, unix.ECONNRESET) || errors.Is(err, unix.EPIPE) {
		return true
	}

	return false
}

const RequestCanceledError = requestCanceledError("RequestCanceled")

type requestCanceledError string

func (e requestCanceledError) Error() string {
	return string(e)
}
