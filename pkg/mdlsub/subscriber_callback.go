package mdlsub

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/stream"
)

const (
	MetricNameSuccess = "ModelEventConsumeSuccess"
	MetricNameFailure = "ModelEventConsumeFailure"
)

type SubscriberModel struct {
	mdl.ModelId
	Shared bool `cfg:"shared"`
}

type SubscriberCallback struct {
	logger      log.Logger
	metric      metric.Writer
	core        SubscriberCore
	sourceModel SubscriberModel
}

func NewSubscriberCallbackFactory(core SubscriberCore, sourceModel SubscriberModel) stream.UntypedConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.UntypedConsumerCallback, error) {
		defaultMetrics := getSubscriberCallbackDefaultMetrics(core.GetModelIds())
		metricWriter := metric.NewWriter(defaultMetrics...)

		callback := &SubscriberCallback{
			logger:      logger,
			metric:      metricWriter,
			core:        core,
			sourceModel: sourceModel,
		}

		return callback, nil
	}
}

func (s *SubscriberCallback) GetModel(attributes map[string]string) any {
	var err error
	var spec *ModelSpecification
	var transformer ModelTransformer

	if spec, err = getModelSpecification(attributes); err != nil {
		return nil
	}

	if transformer, err = s.core.GetTransformer(spec); err != nil {
		return nil
	}

	return transformer.getInput()
}

func (s *SubscriberCallback) GetSchemaSettings() (*stream.SchemaSettings, error) {
	transformersMap, err := s.core.GetTransformersForModel(s.sourceModel.ModelId)
	if err != nil {
		return nil, err
	}

	var schemaSettings *stream.SchemaSettings

	for _, transformer := range transformersMap {
		schemaSettings, err = transformer.getSchemaSettings()
		if err != nil {
			return nil, err
		}

		if schemaSettings != nil && len(transformersMap) > 1 {
			return nil, fmt.Errorf("there should be only one transformer per input model when using the schema registry")
		}
	}

	return schemaSettings, nil
}

func (s *SubscriberCallback) Consume(ctx context.Context, input any, attributes map[string]string) (ack bool, err error) {
	var model Model
	var spec *ModelSpecification
	var transformer ModelTransformer
	var output Output

	if spec, err = getModelSpecification(attributes); err != nil {
		return false, fmt.Errorf("can not read model specifications from the message attributes: %w", err)
	}

	defer func() {
		s.writeMetric(ctx, err, spec)
	}()

	logger := s.logger.WithFields(log.Fields{
		"modelId": spec.ModelId,
		"type":    spec.CrudType,
		"version": spec.Version,
	})

	if transformer, err = s.core.GetTransformer(spec); err != nil {
		return false, err
	}

	if model, err = transformer.transform(ctx, input); err != nil {
		if IsDelayOpError(err) {
			logger.Info(ctx, "delaying %s op for subscription for modelId %s and version %d: %s", spec.CrudType, spec.ModelId, spec.Version, err.Error())

			return false, nil
		}

		return false, err
	}

	if model == nil {
		logger.Info(ctx, "skipping %s op for subscription for modelId %s and version %d", spec.CrudType, spec.ModelId, spec.Version)

		return true, nil
	}

	if output, err = s.core.GetOutput(spec); err != nil {
		return false, err
	}

	err = output.Persist(ctx, model, spec.CrudType)
	if err != nil {
		return false, fmt.Errorf("can not persist subscription of model %s and version %d: %w", spec.ModelId, spec.Version, err)
	}

	logger.Info(ctx, "persisted %s op for subscription for modelId %s and version %d with id %v", spec.CrudType, spec.ModelId, spec.Version, model.GetId())

	return true, nil
}

func (s *SubscriberCallback) writeMetric(ctx context.Context, err error, spec *ModelSpecification) {
	metricName := MetricNameSuccess

	if err != nil {
		metricName = MetricNameFailure
	}

	s.metric.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"ModelId": spec.ModelId,
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})
}

func getSubscriberCallbackDefaultMetrics(modelIds []string) []*metric.Datum {
	defaults := make([]*metric.Datum, 0)

	for _, modelId := range modelIds {
		success := &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNameSuccess,
			Dimensions: map[string]string{
				"ModelId": modelId,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		}

		failure := &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNameFailure,
			Dimensions: map[string]string{
				"ModelId": modelId,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		}

		defaults = append(defaults, success, failure)
	}

	return defaults
}
