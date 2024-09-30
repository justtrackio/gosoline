package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/justtrackio/gosoline/pkg/exec"
)

func init() {
	exec.AddRequestCancelCheck(IsAwsErrorCodeRequestCanceled)
}

func IsAwsErrorCodeRequestCanceled(err error) bool {
	return retry.NoRetryCanceledError{}.IsErrorRetryable(err) == aws.FalseTernary
}
