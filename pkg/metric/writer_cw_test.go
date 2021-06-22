package metric_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/jonboulle/clockwork"
	"testing"
	"time"
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

func buildMocksAndWrite(now time.Time, metricTimeStamp time.Time) *cloudMocks.CloudWatchAPI {
	clock := clockwork.NewFakeClockAt(now)

	logger := logMocks.NewLoggerMockedAll()
	cwClient := new(cloudMocks.CloudWatchAPI)

	cwClient.On("PutMetricData", &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("my/test/namespace/app"),
		MetricData: []*cloudwatch.MetricDatum{{
			MetricName: aws.String("my-test-metric-name"),
			Dimensions: []*cloudwatch.Dimension{
				{
					Name:  aws.String("d1"),
					Value: aws.String("a"),
				},
			},
			Timestamp: aws.Time(metricTimeStamp),
			Value:     aws.Float64(3.4),
			Unit:      aws.String(metric.UnitCount),
		}},
	}).Return(nil, nil)

	mo := metric.NewCwWriterWithInterfaces(logger, clock, cwClient, &metric.Settings{
		AppId: cfg.AppId{
			Project:     "my",
			Environment: "test",
			Family:      "namespace",
			Application: "app",
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
