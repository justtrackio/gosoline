package stream_test

import (
	"context"
	gosoAws "github.com/applike/gosoline/pkg/cloud/aws"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestWriter_WriteEvents(t *testing.T) {
	kinesisClient := new(cloudMocks.KinesisAPI)

	successfulRecordOutput := &kinesis.PutRecordsOutput{Records: []*kinesis.PutRecordsResultEntry{}}
	exec := gosoAws.NewTestableExecutor([]gosoAws.TestExecution{{
		Output: successfulRecordOutput,
		Err:    nil,
	}})

	logger := monMocks.NewLoggerMockedAll()
	writer := stream.NewKinesisOutputWithInterfaces(logger, kinesisClient, exec, &stream.KinesisOutputSettings{
		StreamName: "streamName",
	})

	batch := []*stream.Message{
		stream.NewMessage("1"),
		stream.NewMessage("2"),
		stream.NewMessage("3"),
	}

	kinesisClient.On("PutRecordsRequest", mock.Anything).Return(&request.Request{}, &kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{{
			ErrorCode: aws.String("error"),
		}},
	}).Once()

	kinesisClient.On("PutRecordsRequest", mock.Anything).Return(
		&request.Request{}, successfulRecordOutput).Once()

	logger.On("WithFields", mock.Anything).Return(logger).Run(func(args mock.Arguments) {
		assert.IsType(t, map[string]interface{}{}, args.Get(0))

		fields, _ := args.Get(0).(map[string]interface{})

		assert.Equal(t, 1, fields["total_records"].(int))
	})

	assert.NotPanics(t, func() {
		err := writer.Write(context.Background(), batch)

		assert.NoError(t, err)
	})
}
