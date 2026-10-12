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
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	uuidMocks "github.com/justtrackio/gosoline/pkg/uuid/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestRecordWriterPutRecords(t *testing.T) {
	mw := metricMocks.NewWriter(t)
	mw.EXPECT().Write(matcher.Context, mock.Anything).Return().Times(2)
	testClock := clock.NewFakeClock(clock.WithNonBlockingSleep)

	ctx := log.AppendContextFields(t.Context(), map[string]any{
		"kinesis_write_request_id": "79db3180-99a9-4157-91c3-a591b9a8f01c",
		"stream_name":              "streamName",
	})

	logger := logMocks.NewLoggerMock(logMocks.WithTestingT(t))
	logger.EXPECT().Warn(matcher.Context, "PutRecords failed %d of %d records with reason: %s: after %d attempts in %s", 1, 3, "1 ProvisionedThroughputExceededException errors", 1, time.Second)
	logger.EXPECT().Warn(matcher.Context, "PutRecords successful after %d attempts in %s", 2, mock.AnythingOfType("time.Duration"))

	uuidGen := uuidMocks.NewUuid(t)
	// kinesis kinesis_write_request_id
	uuidGen.EXPECT().NewV4().Return("79db3180-99a9-4157-91c3-a591b9a8f01c").Once()
	// kinesis PartitionKey
	uuidGen.EXPECT().NewV4().Return("ee080b0b-faae-40c2-8959-0f8f2b6d1b06").Once()
	uuidGen.EXPECT().NewV4().Return("51b873fc-8086-4b39-8a68-bead0102cdf0").Once()
	// batch_id
	uuidGen.EXPECT().NewV4().Return("2ac1ed74-7c44-4312-b6da-cabe7b709224").Once()

	kinesisClient := gosoKinesisMocks.NewClient(t)
	kinesisClient.EXPECT().PutRecords(ctx, &kinesis.PutRecordsInput{
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
	}).Run(func(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) {
		testClock.Advance(time.Second)
	}).Return(&kinesis.PutRecordsOutput{
		Records: []types.PutRecordsResultEntry{
			{
				ErrorCode: nil,
			},
			{
				ErrorCode: aws.String("ProvisionedThroughputExceededException"),
			},
			{
				ErrorCode: nil,
			},
		},
		FailedRecordCount: aws.Int32(1),
	}, nil).Once()
	kinesisClient.EXPECT().PutRecords(ctx, &kinesis.PutRecordsInput{
		Records: []types.PutRecordsRequestEntry{
			{
				Data:         []byte("2"),
				PartitionKey: aws.String("541c78c0-afc7-440f-b8a3-d2e49fb1ba4c"),
			},
		},
		StreamName: aws.String("streamName"),
	}).Run(func(ctx context.Context, params *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) {
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
		Backoff: exec.BackoffSettings{
			InitialInterval: time.Millisecond,
			MaxInterval:     time.Millisecond,
		},
	}, "streamName")

	batch := []*gosoKinesis.Record{
		{
			Data: []byte("1"),
		},
		{
			Data:         []byte("2"),
			PartitionKey: aws.String("541c78c0-afc7-440f-b8a3-d2e49fb1ba4c"),
		},
		{
			Data: []byte("3"),
		},
	}

	err := writer.PutRecords(t.Context(), batch)
	assert.NoError(t, err)
}

func TestRecordWriterPutRecordsExportsMetricSemantics(t *testing.T) {
	const streamName = "streamName"

	reason := "ProvisionedThroughputExceededException"
	record := func(id string) *gosoKinesis.Record {
		return &gosoKinesis.Record{
			Data:         []byte(id),
			PartitionKey: aws.String("partition-" + id),
		}
	}
	requestRecord := func(id string) types.PutRecordsRequestEntry {
		return types.PutRecordsRequestEntry{
			Data:         []byte(id),
			PartitionKey: aws.String("partition-" + id),
		}
	}
	failedResult := types.PutRecordsResultEntry{ErrorCode: aws.String(reason)}

	type attempt struct {
		records []types.PutRecordsRequestEntry
		results []types.PutRecordsResultEntry
	}
	tests := []struct {
		name               string
		records            []*gosoKinesis.Record
		attempts           []attempt
		expectedSuccess    float64
		expectedFailed     float64
		expectedReason     string
		expectedBatchCount uint64
		expectedBatchSum   float64
		expectedBatchMean  float64
	}{
		{
			name:    "mixed results with retry failures",
			records: []*gosoKinesis.Record{record("1"), record("2"), record("3")},
			attempts: []attempt{
				{
					records: []types.PutRecordsRequestEntry{requestRecord("1"), requestRecord("2"), requestRecord("3")},
					results: []types.PutRecordsResultEntry{{}, failedResult, {}},
				},
				{
					records: []types.PutRecordsRequestEntry{requestRecord("2")},
					results: []types.PutRecordsResultEntry{failedResult},
				},
				{
					records: []types.PutRecordsRequestEntry{requestRecord("2")},
					results: []types.PutRecordsResultEntry{{}},
				},
			},
			expectedSuccess:    3,
			expectedFailed:     2,
			expectedReason:     "1 ProvisionedThroughputExceededException errors",
			expectedBatchCount: 3,
			expectedBatchSum:   5,
			expectedBatchMean:  5.0 / 3.0,
		},
		{
			name:    "failure-only first attempt",
			records: []*gosoKinesis.Record{record("1"), record("2")},
			attempts: []attempt{
				{
					records: []types.PutRecordsRequestEntry{requestRecord("1"), requestRecord("2")},
					results: []types.PutRecordsResultEntry{failedResult, failedResult},
				},
				{
					records: []types.PutRecordsRequestEntry{requestRecord("1"), requestRecord("2")},
					results: []types.PutRecordsResultEntry{{}, {}},
				},
			},
			expectedSuccess:    2,
			expectedFailed:     2,
			expectedReason:     "2 ProvisionedThroughputExceededException errors",
			expectedBatchCount: 2,
			expectedBatchSum:   4,
			expectedBatchMean:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() {
				require.NoError(t, provider.Shutdown(t.Context()))
			})

			metricWriter := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("record-writer-test"))
			testClock := clock.NewFakeClock(clock.WithNonBlockingSleep)

			uuidGen := uuidMocks.NewUuid(t)
			uuidGen.EXPECT().NewV4().Return("write-request-id").Once()
			uuidGen.EXPECT().NewV4().Return("batch-id").Once()

			kinesisClient := gosoKinesisMocks.NewClient(t)
			for _, attempt := range tt.attempts {
				kinesisClient.EXPECT().PutRecords(matcher.Context, &kinesis.PutRecordsInput{
					Records:    attempt.records,
					StreamName: aws.String(streamName),
				}).Return(&kinesis.PutRecordsOutput{Records: attempt.results}, nil).Once()
			}

			writer := gosoKinesis.NewRecordWriterWithInterfaces(logger, metricWriter, testClock, uuidGen, kinesisClient, &gosoKinesis.RecordWriterSettings{
				StreamName: streamName,
				Backoff: exec.BackoffSettings{
					InitialInterval: time.Millisecond,
					MaxInterval:     time.Millisecond,
				},
			}, streamName)

			require.NoError(t, writer.PutRecords(t.Context(), tt.records))

			var exported metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(t.Context(), &exported))
			require.Len(t, exported.ScopeMetrics, 1)

			metricsByName := make(map[string]metricdata.Metrics, len(exported.ScopeMetrics[0].Metrics))
			for _, exportedMetric := range exported.ScopeMetrics[0].Metrics {
				metricsByName[exportedMetric.Name] = exportedMetric
			}

			require.Len(t, metricsByName, 2)
			sentMessages, ok := metricsByName["gosoline.cloud.aws.kinesis.sent.messages"]
			require.True(t, ok)
			assert.Equal(t, "{message}", sentMessages.Unit)
			sentCounter, ok := sentMessages.Data.(metricdata.Sum[float64])
			require.True(t, ok)
			assert.Equal(t, metricdata.CumulativeTemporality, sentCounter.Temporality)
			assert.True(t, sentCounter.IsMonotonic)
			require.Len(t, sentCounter.DataPoints, 2)

			var successfulRecords, failedRecords float64
			var successPointSeen, failurePointSeen bool
			for _, point := range sentCounter.DataPoints {
				attributes := recordWriterMetricAttributes(point.Attributes)
				assert.Equal(t, streamName, attributes["stream.name"])

				if errorType, hasErrorType := attributes[metric.DimensionErrorType]; hasErrorType {
					assert.Equal(t, tt.expectedReason, errorType)
					assert.Equal(t, map[string]string{
						"stream.name":             streamName,
						metric.DimensionErrorType: tt.expectedReason,
					}, attributes)
					failedRecords = point.Value
					failurePointSeen = true

					continue
				}

				assert.Equal(t, map[string]string{"stream.name": streamName}, attributes)
				successfulRecords = point.Value
				successPointSeen = true
			}
			require.True(t, successPointSeen)
			require.True(t, failurePointSeen)
			assert.Equal(t, tt.expectedSuccess, successfulRecords)
			assert.Equal(t, tt.expectedFailed, failedRecords)

			batchRecords, ok := metricsByName["gosoline.cloud.aws.kinesis.batch.records"]
			require.True(t, ok)
			assert.Equal(t, "{record}", batchRecords.Unit)
			batchHistogram, ok := batchRecords.Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, batchHistogram.DataPoints, 1)
			batchPoint := batchHistogram.DataPoints[0]
			assert.Equal(t, map[string]string{"stream.name": streamName}, recordWriterMetricAttributes(batchPoint.Attributes))
			assert.Equal(t, tt.expectedBatchCount, batchPoint.Count)
			assert.Equal(t, tt.expectedBatchSum, batchPoint.Sum)
			assert.InDelta(t, tt.expectedBatchMean, batchPoint.Sum/float64(batchPoint.Count), 1e-9)
		})
	}
}

func recordWriterMetricAttributes(attributes attribute.Set) map[string]string {
	values := make(map[string]string, attributes.Len())
	for _, value := range attributes.ToSlice() {
		values[string(value.Key)] = value.Value.AsString()
	}

	return values
}
