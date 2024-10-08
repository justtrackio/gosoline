package aws_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
)

func TestBackoffExecutor_Execute(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	executions := 0

	executor := aws.NewBackoffExecutorWithSender(logger, &exec.ExecutableResource{
		Type: "ddb",
		Name: "test-table",
	}, &exec.BackoffSettings{}, func(req *request.Request) (*http.Response, error) {
		executions++
		switch executions {
		case 1:
			return nil, &aws.InvalidStatusError{}
		case 2:
			return &http.Response{
				Status:     "Internal Server Error",
				StatusCode: http.StatusInternalServerError,
			}, fmt.Errorf("net/http: request canceled")
		case 3:
			return &http.Response{
				Status:     "Internal Server Error",
				StatusCode: http.StatusInternalServerError,
			}, nil
		default:
			*req.Data.(*[]string) = []string{"foo"}
			return &http.Response{
				Status:     "Ok",
				StatusCode: http.StatusOK,
			}, nil
		}
	})

	out, err := executor.Execute(context.Background(), func() (*request.Request, interface{}) {
		req := &request.Request{
			HTTPRequest: &http.Request{},
		}
		out := &[]string{}
		req.Data = out

		return req, out
	})

	assert.NoError(t, err)
	assert.Equal(t, &[]string{"foo"}, out)
}
