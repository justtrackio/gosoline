package metric_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
)

func TestOutput_Write(t *testing.T) {
	timestamp := time.Unix(1549283566, 0)
	cwClient := buildMocksAndWrite(timestamp, timestamp)

	cwClient.AssertExpectations(t)
}

func TestOutput_Write_OutOfRange(t *testing.T) {
	now := time.Unix(1549283566, 0)
	timestamp := now.Add(-2 * 7 * 24 * time.Hour)

	cwClient := buildMocksAndWrite(now, timestamp)

	cwClient.AssertNotCalled(t, "PutMetricData", "data should be out of range")
}

func buildMocksAndWrite(now time.Time, metricTimeStamp time.Time) *cloudwatchMocks.Client {
	testClock := clock.NewFakeClockAt(now)

	logger := logMocks.NewLoggerMockedAll()
	cwClient := new(cloudwatchMocks.Client)

	cwClient.On("PutMetricData", context.Background(), &cloudwatch.PutMetricDataInput{
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

	mo := metric.NewCwWriterWithInterfaces(logger, testClock, cwClient, &metric.Settings{
		AppId: cfg.AppId{
			Project:     "my",
			Environment: "test",
			Family:      "namespace",
			Group:       "grp",
			Application: "app",
		},
		Cloudwatch: metric.Cloudwatch{
			Naming: metric.NamingSettings{
				Pattern: "{project}/{env}/{family}/{group}/{app}",
			},
		},
		Enabled: true,
	})

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

	return cwClient
}
