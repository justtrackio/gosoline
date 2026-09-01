package metric_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
)

func TestOutput_Write(t *testing.T) {
	timestamp := time.Unix(1549283566, 0)
	buildMocksAndWrite(t, t.Context(), timestamp, timestamp, true)
}

func TestOutput_Write_OutOfRange(t *testing.T) {
	now := time.Unix(1549283566, 0)
	timestamp := now.Add(-2 * 7 * 24 * time.Hour)

	buildMocksAndWrite(t, t.Context(), now, timestamp, false)
}

func TestOutput_WriteWithCancelledContext(t *testing.T) {
	now := time.Unix(1549283566, 0)
	timestamp := now

	deadCtx, cancel := context.WithCancel(t.Context())
	cancel()

	buildMocksAndWrite(t, deadCtx, now, timestamp, true)
}

func buildMocksAndWrite(t *testing.T, ctx context.Context, now time.Time, metricTimeStamp time.Time, shouldPutMetricData bool) {
	testClock := clock.NewFakeClockAt(now)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll)
	cwClient := cloudwatchMocks.NewClient(t)

	if shouldPutMetricData {
		cwClient.EXPECT().PutMetricData(matcher.Context, &cloudwatch.PutMetricDataInput{
			Namespace: aws.String("my/test/namespace/grp/app"),
			MetricData: []types.MetricDatum{{
				MetricName: aws.String("my-test-metric-name"),
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("d1"),
						Value: aws.String("a"),
					},
				},
				Timestamp: aws.Time(metricTimeStamp),
				Value:     aws.Float64(3.4),
				Unit:      metric.UnitCount,
			}},
		}).Return(nil, nil)
	}

	mo := metric.NewCloudwatchWriterWithInterfaces(
		logger,
		testClock,
		cwClient,
		"my/test/namespace/grp/app",
		10*time.Second,
	)

	data := metric.Data{
		{
			Priority:   metric.PriorityHigh,
			Timestamp:  metricTimeStamp,
			MetricName: "my-test-metric-name",
			Dimensions: map[string]string{
				"d1": "a",
			},
			Unit:  metric.UnitCount,
			Value: 3.4,
		},
	}

	mo.Write(ctx, data)
}

// TestOutput_WriteRendersCanonicalNames pins down the CloudWatch rendering of a canonical namespace
// and leaf: one PascalCase name without a Gosoline prefix, the resolved base unit, and an unscaled
// value, so CloudWatch keeps reporting durations in milliseconds.
func TestOutput_WriteRendersCanonicalNames(t *testing.T) {
	tests := map[string]struct {
		datum        *metric.Datum
		expectedName string
		expectedUnit types.StandardUnit
		expectedVal  float64
	}{
		"semantic convention name": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "http.server",
				MetricName: "request.duration",
				Unit:       metric.UnitMilliseconds,
				Value:      250,
			},
			expectedName: "HttpServerRequestDuration",
			expectedUnit: metric.UnitMilliseconds,
			expectedVal:  250,
		},
		"multi word component": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "aws.kinesis.shard",
				MetricName: "acquire.delay",
				Unit:       metric.UnitSeconds,
				Value:      3,
			},
			expectedName: "AwsKinesisShardAcquireDelay",
			expectedUnit: metric.UnitSeconds,
			expectedVal:  3,
		},
		"custom aggregation unit resolves to its base unit": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "conc.scheduler",
				MetricName: "task.delay",
				Unit:       metric.UnitMillisecondsAverage,
				Value:      250,
			},
			expectedName: "ConcSchedulerTaskDelay",
			expectedUnit: metric.UnitMilliseconds,
			expectedVal:  250,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			now := time.Unix(1549283566, 0)
			tt.datum.Timestamp = now

			logger := logMocks.NewLoggerMock(logMocks.WithMockAll)
			cwClient := cloudwatchMocks.NewClient(t)

			cwClient.EXPECT().PutMetricData(matcher.Context, &cloudwatch.PutMetricDataInput{
				Namespace: aws.String("my/test/namespace"),
				MetricData: []types.MetricDatum{{
					MetricName: aws.String(tt.expectedName),
					Dimensions: []types.Dimension{},
					Timestamp:  aws.Time(now),
					Value:      aws.Float64(tt.expectedVal),
					Unit:       tt.expectedUnit,
				}},
			}).Return(nil, nil)

			writer := metric.NewCloudwatchWriterWithInterfaces(
				logger,
				clock.NewFakeClockAt(now),
				cwClient,
				"my/test/namespace",
				10*time.Second,
			)

			writer.Write(t.Context(), metric.Data{tt.datum})
		})
	}
}
