package aws

import (
	"errors"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
)

func init() {
	exec.AddRequestCancelCheck(IsAwsErrorCodeRequestCanceled)
}

func IsAwsError(err error, awsCode string) bool {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) {
		return awsErr.Code() == awsCode
	}

	return false
}

func IsAwsErrorCodeRequestCanceled(err error) bool {
	if IsAwsError(err, request.CanceledErrorCode) {
		return true
	}

	return false
}

func CheckConnectionError(_ interface{}, err error) exec.ErrorType {
	if IsConnectionError(err) {
		return exec.ErrorRetryable
	}

	return exec.ErrorUnknown
}

func IsConnectionError(err error) bool {
	var awsErr awserr.Error

	if errors.As(err, &awsErr) {
		err = awsErr.OrigErr()
	}

	return exec.IsConnectionError(awsErr)
}

func CheckErrorRetryable(_ interface{}, err error) exec.ErrorType {
	if request.IsErrorRetryable(err) {
		return exec.ErrorRetryable
	}

	return exec.ErrorUnknown
}

func CheckErrorThrottle(_ interface{}, err error) exec.ErrorType {
	if request.IsErrorThrottle(err) {
		return exec.ErrorRetryable
	}

	return exec.ErrorUnknown
}
