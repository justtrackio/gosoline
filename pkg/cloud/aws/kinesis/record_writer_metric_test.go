package kinesis

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testRecordWriterMetricType = "record-writer-default-export-test"

func TestRecordWriterDefaultMetricReachesCloudWatchThroughDaemon(t *testing.T) {
	const streamName = "record-writer-default-export-test-stream"

	metric.NewWriter(metricNamespaceCloudAwsKinesis, getRecordWriterDefaultMetrics(streamName)...)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	cloudwatchClient := cloudwatchMocks.NewClient(t)
	var exported []types.MetricDatum

	cloudwatchClient.EXPECT().PutMetricData(matcher.Context, mock.Anything).
		Run(func(_ context.Context, input *cloudwatch.PutMetricDataInput, _ ...func(*cloudwatch.Options)) {
			exported = append(exported, input.MetricData...)
		}).Return(nil, nil).Maybe()

	metric.RegisterWriterFactory(testRecordWriterMetricType, func(_ context.Context, _ cfg.Config, logger log.Logger) (metric.Writer, error) {
		return metric.NewCloudwatchWriterWithInterfaces(
			logger,
			clock.NewFakeClockAt(time.Now().Add(time.Second)),
			cloudwatchClient,
			"test",
			0,
		), nil
	})

	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "kinesis_metric_daemon_test",
			"tags": map[string]any{
				"project": "justtrack",
				"family":  "gosoline",
				"group":   "gosoline",
			},
		},
		"metric": map[string]any{
			"enabled":  true,
			"interval": "1s",
			"writers":  []string{testRecordWriterMetricType},
			"writer_settings": map[string]any{
				testRecordWriterMetricType: map[string]any{
					"aggregate": true,
				},
			},
		},
	})

	ctx := appctx.WithContainer(t.Context())
	daemon, err := metric.NewDaemonModule(ctx, config, logger)
	require.NoError(t, err)
	require.NotNil(t, daemon)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	require.NoError(t, daemon.Run(canceledCtx))

	var batchRecords *types.MetricDatum
	for i := range exported {
		if aws.ToString(exported[i].MetricName) == "CloudAwsKinesisBatchRecords" {
			batchRecords = &exported[i]

			break
		}
	}

	require.NotNil(t, batchRecords, "batch.records default must reach the CloudWatch exporter")
	assert.Equal(t, metric.UnitCount, batchRecords.Unit)
	assert.Equal(t, float64(0), aws.ToFloat64(batchRecords.Value))
	require.Len(t, batchRecords.Dimensions, 1)
	assert.Equal(t, dimensionStream, aws.ToString(batchRecords.Dimensions[0].Name))
	assert.Equal(t, streamName, aws.ToString(batchRecords.Dimensions[0].Value))
}
