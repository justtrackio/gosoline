package db_repo

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNameNotifySuccess = "ModelEventNotifySuccess"
	metricNameNotifyFailure = "ModelEventNotifyFailure"
)

var NotificationTypes = []string{Create, Update, Delete}

type (
	Publisher interface {
		Publish(ctx context.Context, typ string, version int, value any, customAttributes ...map[string]string) error
	}
	NotificationMap[K mdl.PossibleIdentifier] map[string][]Notifier[K]
	Notifier[K mdl.PossibleIdentifier]        interface {
		Send(ctx context.Context, notificationType string, value ModelBased[K]) error
	}
)

type notifier struct {
	logger  log.Logger
	metric  metric.Writer
	modelId mdl.ModelId
	version int
}

func newNotifier(logger log.Logger, modelId mdl.ModelId, version int) notifier {
	defaults := getDefaultNotifierMetrics(modelId)
	mtr := metric.NewWriter(defaults...)

	return notifier{
		logger:  logger,
		metric:  mtr,
		modelId: modelId,
		version: version,
	}
}

func (n *notifier) writeMetric(ctx context.Context, err error) {
	metricName := "ModelEventNotifySuccess"

	if err != nil {
		metricName = "ModelEventNotifyFailure"
	}

	n.metric.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"ModelId": n.modelId.String(),
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getDefaultNotifierMetrics(modelId mdl.ModelId) []*metric.Datum {
	return []*metric.Datum{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifySuccess,
			Dimensions: map[string]string{
				"ModelId": modelId.String(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifyFailure,
			Dimensions: map[string]string{
				"ModelId": modelId.String(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
