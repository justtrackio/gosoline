package kinesis_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	gosoKinesisMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRecordWriterPutRecords(t *testing.T) {
	ctx := log.AppendLoggerContextField(context.Background(), map[string]interface{}{
		"kinesis_write_request_id": "79db3180-99a9-4157-91c3-a591b9a8f01c",
		"stream_name":              "streamName",
	})

	logger := logMocks.NewLoggerMock()
	logger.On("Warn", "%d / %d records failed, retrying", 1, 3)
	logger.On("Warn", "PutRecords successful after %d retries in %s", 1, 2*time.Second)

	mw := metricMocks.NewWriterMockedAll()

	testClock := clock.NewFakeClock()

	uuidGen := new(uuidMocks.Uuid)
	uuidGen.On("NewV4").Return("79db3180-99a9-4157-91c3-a591b9a8f01c").Once()
	uuidGen.On("NewV4").Return("ee080b0b-faae-40c2-8959-0f8f2b6d1b06").Once()
	uuidGen.On("NewV4").Return("541c78c0-afc7-440f-b8a3-d2e49fb1ba4c").Once()
	uuidGen.On("NewV4").Return("51b873fc-8086-4b39-8a68-bead0102cdf0").Once()

	kinesisClient := new(gosoKinesisMocks.Client)
	kinesisClient.On("PutRecords", ctx, &kinesis.PutRecordsInput{
		Records: []types.PutRecordsRequestEntry{
			{
				Data:         []byte("1"),
				PartitionKey: aws.String("ee080b0b-faae-40c2-8959-0f8f2b6d1b06"),
			},
			{
				Data:         []byte("2"),
				PartitionKey: aws.String("541c78c0-afc7-440f-b8a3-d2e49fb1ba4c"),
			},
			{
				Data:         []byte("3"),
				PartitionKey: aws.String("51b873fc-8086-4b39-8a68-bead0102cdf0"),
			},
		},
		StreamName: aws.String("streamName"),
	}).Run(func(args mock.Arguments) {
		testClock.Advance(time.Second)
	}).Return(&kinesis.PutRecordsOutput{
		Records: []types.PutRecordsResultEntry{
			{
				ErrorCode: nil,
			},
			{
				ErrorCode: aws.String("throttling"),
			},
			{
				ErrorCode: nil,
			},
		},
		FailedRecordCount: aws.Int32(1),
	}, nil).Once()
	kinesisClient.On("PutRecords", ctx, &kinesis.PutRecordsInput{
		Records: []types.PutRecordsRequestEntry{
			{
				Data:         []byte("2"),
				PartitionKey: aws.String("541c78c0-afc7-440f-b8a3-d2e49fb1ba4c"),
			},
		},
		StreamName: aws.String("streamName"),
	}).Run(func(args mock.Arguments) {
		testClock.Advance(time.Second)
	}).Return(&kinesis.PutRecordsOutput{
		Records: []types.PutRecordsResultEntry{
			{
				ErrorCode: nil,
			},
		},
		FailedRecordCount: aws.Int32(0),
	}, nil).Once()

	writer := gosoKinesis.NewRecordWriterWithInterfaces(logger, mw, testClock, uuidGen, kinesisClient, &gosoKinesis.RecordWriterSettings{
		StreamName: "streamName",
	})

	batch := [][]byte{
		[]byte("1"),
		[]byte("2"),
		[]byte("3"),
	}

	err := writer.PutRecords(context.Background(), batch)
	assert.NoError(t, err)

	logger.AssertExpectations(t)
	uuidGen.AssertExpectations(t)
	kinesisClient.AssertExpectations(t)
}
