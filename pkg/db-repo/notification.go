package db_repo

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/metric"
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
	logger      log.Logger
	metric      metric.Writer
	encoder     stream.MessageEncoder
	output      stream.Output
	modelId     mdl.ModelId
	version     int
	transformer mdl.TransformerResolver
}

func NewBaseNotifier(logger log.Logger, output stream.Output, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *baseNotifier {
	defaults := getDefaultNotifierMetrics(modelId)
	mtr := metric.NewDaemonWriter(defaults...)

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})

	return &baseNotifier{
		logger:      logger,
		metric:      mtr,
		encoder:     encoder,
		output:      output,
		modelId:     modelId,
		version:     version,
		transformer: transformer,
	}
}

func (n baseNotifier) Send(ctx context.Context, notificationType string, value ModelBased) error {
	logger := n.logger.WithContext(ctx)
	modelId := n.modelId.String()

	out := n.transformer("api", n.version, value)

	msg, err := n.encoder.Encode(ctx, out, map[string]interface{}{
		"type":    notificationType,
		"version": n.version,
		"modelId": modelId,
	})

	if err != nil {
		return fmt.Errorf("can not encode notification message: %w", err)
	}

	err = n.output.WriteOne(ctx, msg)

	if exec.IsRequestCanceled(err) {
		logger.Info("request canceled while executing notification on %s for model %s with id %d", notificationType, modelId, *value.GetId())
		n.writeMetric(err)
		return err
	}

	if err != nil {
		logger.Error("error executing notification on %s for model %s with id %d: %w", notificationType, modelId, *value.GetId(), err)
		n.writeMetric(err)
		return err
	}

	logger.Info("sent on %s successful for model %s with id %d", notificationType, modelId, *value.GetId())
	n.writeMetric(nil)

	return nil
}

func (n baseNotifier) writeMetric(err error) {
	metricName := "ModelEventNotifySuccess"

	if err != nil {
		metricName = "ModelEventNotifyFailure"
	}

	n.metric.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Application": n.modelId.Application,
			"ModelId":     n.modelId.String(),
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
				"Application": modelId.Application,
				"ModelId":     modelId.String(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameNotifyFailure,
			Dimensions: map[string]string{
				"Application": modelId.Application,
				"ModelId":     modelId.String(),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
