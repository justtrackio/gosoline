package stream_test

import (
	"context"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestWriter_WriteEvents(t *testing.T) {
	kinesisClient := new(cloudMocks.KinesisAPI)
	exec := gosoAws.NewTestableExecutor(&kinesisClient.Mock)

	failureRecordOutput := &kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{{
			ErrorCode: aws.String("error"),
		}},
	}
	exec.ExpectExecution("PutRecordsRequest", mock.AnythingOfType("*kinesis.PutRecordsInput"), failureRecordOutput, nil)

	successfulRecordOutput := &kinesis.PutRecordsOutput{Records: []*kinesis.PutRecordsResultEntry{}}
	exec.ExpectExecution("PutRecordsRequest", mock.AnythingOfType("*kinesis.PutRecordsInput"), successfulRecordOutput, nil)

	logger := monMocks.NewLoggerMockedAll()
	writer := stream.NewKinesisOutputWithInterfaces(logger, kinesisClient, exec, &stream.KinesisOutputSettings{
		StreamName: "streamName",
	})

	batch := []*stream.Message{
		stream.NewMessage("1"),
		stream.NewMessage("2"),
		stream.NewMessage("3"),
	}

	assert.NotPanics(t, func() {
		err := writer.Write(context.Background(), batch)

		assert.NoError(t, err)
	})
}
