package db_repo

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNameNotifications = "model_event.notifications"
	metricNameNotifyErrors  = "model_event.notify.errors"
)

var NotificationTypes = []string{Create, Update, Delete}

type (
	Publisher interface {
		Publish(ctx context.Context, typ string, version int, value any, customAttributes ...map[string]string) error
	}
	NotificationMap map[string][]Notifier
	Notifier        interface {
		Send(ctx context.Context, notificationType string, value ModelBased) error
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
	mtr := metric.NewWriter(metric.NamespaceDbRepo, defaults...)

	return notifier{
		logger:  logger,
		metric:  mtr,
		modelId: modelId,
		version: version,
	}
}

func (n *notifier) writeMetric(ctx context.Context, err error) {
	datum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricNameNotifications,
		Dimensions: map[string]string{
			metric.DimensionModelId: n.modelId.String(),
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
		Kind:  metric.KindCounter.Build(),
	}

	if err != nil {
		datum.MetricName = metricNameNotifyErrors
		datum.Dimensions[metric.DimensionErrorType] = metric.ErrorType(err)
	}

	n.metric.WriteOne(ctx, datum)
}

func getDefaultNotifierMetrics(modelId mdl.ModelId) []*metric.Datum {
	return []*metric.Datum{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifications,
			Dimensions: map[string]string{
				metric.DimensionModelId: modelId.String(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifyErrors,
			Dimensions: map[string]string{
				metric.DimensionModelId:   modelId.String(),
				metric.DimensionErrorType: metric.DimensionDefault,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		},
	}
}
