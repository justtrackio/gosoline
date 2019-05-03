package stream_test

import (
	"context"
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

	logger := monMocks.NewLoggerMockedAll()
	writer := stream.NewKinesisOutputWithInterfaces(logger, kinesisClient, "streamName")

	batch := []*stream.Message{{Body: "1"}, {Body: "2"}, {Body: "3"}}

	kinesisClient.On("PutRecords", mock.Anything).Return(&kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{{
			ErrorCode: aws.String("error"),
		}},
	}, nil).Once()

	kinesisClient.On("PutRecords", mock.Anything).Return(&kinesis.PutRecordsOutput{
		Records: []*kinesis.PutRecordsResultEntry{{}},
	}, nil).Once()

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
