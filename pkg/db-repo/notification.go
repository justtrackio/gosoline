package db_repo

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
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
	NotificationMap map[string][]Notifier
	Notifier        interface {
		Send(ctx context.Context, notificationType string, value ModelBased) error
	}
)

type notifier struct {
	logger        log.Logger
	metric        metric.Writer
	modelIdString string
	version       int
}

func newNotifier(logger log.Logger, modelIdString string, version int) notifier {
	defaults := getDefaultNotifierMetrics(modelIdString)
	mtr := metric.NewWriter(defaults...)

	return notifier{
		logger:        logger,
		metric:        mtr,
		modelIdString: modelIdString,
		version:       version,
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
			"ModelId": n.modelIdString,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getDefaultNotifierMetrics(modelIdString string) []*metric.Datum {
	return []*metric.Datum{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifySuccess,
			Dimensions: map[string]string{
				"ModelId": modelIdString,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifyFailure,
			Dimensions: map[string]string{
				"ModelId": modelIdString,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
