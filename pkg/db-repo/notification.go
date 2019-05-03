package db_repo

import (
	"context"
	"encoding/json"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

const (
	metricNameNotifySuccess = "ModelEventNotifySuccess"
	metricNameNotifyFailure = "ModelEventNotifyFailure"
)

var NotificationTypes = []string{Create, Update, Delete}

type NotificationMap map[string][]Notifier
type Notifier interface {
	Send(ctx context.Context, notificationType string, value ModelBased) error
}

type baseNotifier struct {
	logger      mon.Logger
	metric      mon.MetricWriter
	output      stream.Output
	modelId     mdl.ModelId
	version     int
	transformer mdl.TransformerResolver
}

func (n baseNotifier) Send(ctx context.Context, notificationType string, value ModelBased) error {
	logger := n.logger.WithContext(ctx)
	modelId := n.modelId.String()

	out := n.transformer("api", n.version, value)
	body, err := json.Marshal(out)

	if err != nil {
		return err
	}

	msg := stream.CreateMessageFromContext(ctx)
	msg.Attributes["type"] = notificationType
	msg.Attributes["version"] = n.version
	msg.Attributes["modelId"] = modelId
	msg.Body = string(body)

	err = n.output.WriteOne(ctx, msg)

	if err != nil {
		logger.Errorf(err, "error executing notification on %s for model %s with id %d", notificationType, modelId, *value.GetId())
		n.writeMetric(err)
		return err
	}

	logger.Infof("sent on %s successful for model %s with id %d", notificationType, modelId, *value.GetId())
	n.writeMetric(nil)

	return nil
}

func (n baseNotifier) writeMetric(err error) {
	metricName := "ModelEventNotifySuccess"

	if err != nil {
		metricName = "ModelEventNotifyFailure"
	}

	n.metric.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Application": n.modelId.Application,
			"ModelId":     n.modelId.String(),
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})
}

func getDefaultNotifierMetrics(modelId mdl.ModelId) []*mon.MetricDatum {
	return []*mon.MetricDatum{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameNotifySuccess,
			Dimensions: map[string]string{
				"Application": modelId.Application,
				"ModelId":     modelId.String(),
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricNameNotifyFailure,
			Dimensions: map[string]string{
				"Application": modelId.Application,
				"ModelId":     modelId.String(),
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}
}
