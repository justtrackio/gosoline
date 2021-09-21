package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/smithy-go"
	"github.com/justtrackio/gosoline/pkg/exec"
)

func init() {
	exec.AddRequestCancelCheck(IsAwsErrorCodeRequestCanceled)
}

type InvalidStatusError struct {
	Status int
}

func (e *InvalidStatusError) Error() string {
	return fmt.Sprintf("http status code: %d", e.Status)
}

func (e *InvalidStatusError) Is(err error) bool {
	_, ok := err.(*InvalidStatusError)

	return ok
}

func (e *InvalidStatusError) As(target interface{}) bool {
	err, ok := target.(*InvalidStatusError)

	if ok && err != nil {
		*err = *e
	}

	return ok
}

func IsInvalidStatusError(err error) bool {
	return errors.Is(err, &InvalidStatusError{})
}

func CheckInvalidStatusError(_ interface{}, err error) exec.ErrorType {
	if IsInvalidStatusError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func IsAwsError(err error, awsCode string) bool {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) {
		return awsErr.Code() == awsCode
	}

	return false
}

func IsAwsErrorCodeRequestCanceled(err error) bool {
	var errCancel *smithy.CanceledError
	if errors.As(err, &errCancel) {
		return true
	}

	return IsAwsError(err, request.CanceledErrorCode)
}

func CheckConnectionError(_ interface{}, err error) exec.ErrorType {
	if IsConnectionError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func IsConnectionError(err error) bool {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) && awsErr.OrigErr() != nil {
		err = awsErr.OrigErr()
	}

	return exec.IsConnectionError(err)
}

func CheckErrorRetryable(_ interface{}, err error) exec.ErrorType {
	if request.IsErrorRetryable(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckErrorThrottle(_ interface{}, err error) exec.ErrorType {
	if request.IsErrorThrottle(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}
