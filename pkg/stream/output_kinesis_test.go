package stream_test

import (
	"context"
	"testing"

	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	"github.com/applike/gosoline/pkg/exec"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWriter_WriteEvents(t *testing.T) {
	kinesisClient := new(cloudMocks.KinesisAPI)
	executor := gosoAws.NewTestableExecutor(&kinesisClient.Mock)
	resource := &exec.ExecutableResource{
		Type: "kinesis.batch",
		Name: "streamName",
	}

	failureRecordOutput := &kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{{
			ErrorCode: aws.String("error"),
		}},
	}
	executor.ExpectExecution("PutRecordsRequest", mock.AnythingOfType("*kinesis.PutRecordsInput"), failureRecordOutput, nil)

	successfulRecordOutput := &kinesis.PutRecordsOutput{Records: []*kinesis.PutRecordsResultEntry{}}
	executor.ExpectExecution("PutRecordsRequest", mock.AnythingOfType("*kinesis.PutRecordsInput"), successfulRecordOutput, nil)

	logger := logMocks.NewLoggerMock()
	logger.On("Warn", "retrying resource %s after error: %s", resource, "1 out of 3 records failed")
	logger.On("Info", "sent request to resource %s successful after %d attempts in %s", resource, 2, mock.AnythingOfType("time.Duration"))

	writer := stream.NewKinesisOutputWithInterfaces(logger, kinesisClient, executor, &stream.KinesisOutputSettings{
		StreamName: "streamName",
	})

	batch := []stream.WritableMessage{
		stream.NewMessage("1"),
		stream.NewMessage("2"),
		stream.NewMessage("3"),
	}

	assert.NotPanics(t, func() {
		err := writer.Write(context.Background(), batch)

		assert.NoError(t, err)
	})
}
