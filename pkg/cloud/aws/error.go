package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/smithy-go"
	"github.com/justtrackio/gosoline/pkg/exec"
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
	var errCancel *smithy.CanceledError
	if errors.As(err, &errCancel) {
		return true
	}

	return IsAwsError(err, request.CanceledErrorCode)
}

type RetryOnClosedNetworkConnection struct{}

func (r RetryOnClosedNetworkConnection) IsErrorRetryable(err error) aws.Ternary {
	if exec.IsUsedClosedConnectionError(err) {
		return aws.TrueTernary
	}

	return aws.UnknownTernary
}
