package metric_test

import (
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
	buildMocksAndWrite(t, timestamp, timestamp, true)
}

func TestOutput_Write_OutOfRange(t *testing.T) {
	now := time.Unix(1549283566, 0)
	timestamp := now.Add(-2 * 7 * 24 * time.Hour)

	buildMocksAndWrite(t, now, timestamp, false)
}

func buildMocksAndWrite(t *testing.T, now time.Time, metricTimeStamp time.Time, shouldPutMetricData bool) {
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

	mo := metric.NewCloudwatchWriterWithInterfaces(logger, testClock, cwClient, "my/test/namespace/grp/app")

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

	mo.Write(data)
}
