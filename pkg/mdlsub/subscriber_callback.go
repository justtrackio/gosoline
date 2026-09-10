package mdlsub

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/stream"
)

const (
	metricNamespace = "mdlsub"

	MetricNameEvents = "events"

	// dimensionOutcome tells apart what a subscriber did with an event. A failed event is told apart by
	// error.type instead, so one metric answers how many events arrived and what became of them.
	dimensionOutcome = "outcome"

	OutcomeApplied = "applied"
	OutcomeSkipped = "skipped"
)

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameEvents, "model events a subscriber received, by outcome and error type")
}

type SubscriberModel struct {
	mdl.ModelId
	Shared bool `cfg:"shared"`
}

type SubscriberCallback struct {
	logger           log.Logger
	metric           metric.Writer
	core             SubscriberCore
	sourceModel      SubscriberModel
	persistGraceTime time.Duration
}

func NewSubscriberCallbackFactory(
	core SubscriberCore,
	sourceModel SubscriberModel,
	persistGraceTime time.Duration,
) stream.UntypedConsumerCallbackFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (stream.UntypedConsumerCallback, error) {
		defaultMetrics := getSubscriberCallbackDefaultMetrics(core.GetModelIds())
		metricWriter := metric.NewWriter(metricNamespace, defaultMetrics...)

		callback := &SubscriberCallback{
			logger:           logger,
			metric:           metricWriter,
			core:             core,
			sourceModel:      sourceModel,
			persistGraceTime: persistGraceTime,
		}

		return callback, nil
	}
}

// NewSubscriberCallbackWithInterfaces creates a SubscriberCallback for testing purposes
func NewSubscriberCallbackWithInterfaces(
	logger log.Logger,
	core SubscriberCore,
	sourceModel SubscriberModel,
) *SubscriberCallback {
	defaultMetrics := getSubscriberCallbackDefaultMetrics(core.GetModelIds())
	metricWriter := metric.NewWriter(metricNamespace, defaultMetrics...)

	return &SubscriberCallback{
		logger:           logger,
		metric:           metricWriter,
		core:             core,
		sourceModel:      sourceModel,
		persistGraceTime: 0,
	}
}

func (s *SubscriberCallback) GetModel(attributes map[string]string) (any, error) {
	spec, err := getModelSpecification(attributes)
	if err != nil {
		return nil, fmt.Errorf("can not read model specifications from the message attributes: %w", err)
	}

	// Validate that the model and version exist
	transformer, err := s.core.GetTransformer(spec)
	if err != nil {
		return nil, err
	}

	return transformer.getInput(), nil
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
		if err != nil {
			s.writeErrorMetric(ctx, spec, err)
		}
	}()

	logger := s.logger.WithFields(log.Fields{
		"modelId": spec.ModelId,
		"type":    spec.CrudType,
		"version": spec.Version,
	})

	if transformer, err = s.core.GetTransformer(spec); err != nil {
		// This should not happen if GetModel was called first, but handle it for safety
		return false, err
	}

	if model, err = transformer.transform(ctx, input); err != nil {
		if IsDelayOpError(err) || exec.IsRequestCanceled(err) {
			logger.Info(
				ctx,
				"delaying %s op for subscription for modelId %s and version %d: %s",
				spec.CrudType,
				spec.ModelId,
				spec.Version,
				err.Error(),
			)

			return false, nil
		}

		return false, err
	}

	if model == nil {
		logger.Info(ctx, "skipping %s op for subscription for modelId %s and version %d", spec.CrudType, spec.ModelId, spec.Version)
		s.writeMetric(ctx, OutcomeSkipped, spec)

		return true, nil
	}

	if output, err = s.core.GetOutput(spec); err != nil {
		return false, err
	}

	ctx, stop := exec.WithDelayedCancelContext(ctx, s.persistGraceTime)
	defer stop()

	err = output.Persist(ctx, model, spec.CrudType)
	if exec.IsRequestCanceled(err) {
		logger.Warn(ctx, "failed to persist subscription for modelId %s and version %d: %s", err)

		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("can not persist subscription of model %s and version %d: %w", spec.ModelId, spec.Version, err)
	}

	logger.Info(
		ctx,
		"persisted %s op for subscription for modelId %s and version %d with id %v",
		spec.CrudType,
		spec.ModelId,
		spec.Version,
		model.GetId(),
	)

	s.writeMetric(ctx, OutcomeApplied, spec)

	return true, nil
}

func (s *SubscriberCallback) writeMetric(ctx context.Context, outcome string, spec *ModelSpecification) {
	s.metric.WriteOne(ctx, eventDatum(spec.ModelId, outcome, metric.DimensionDefault, 1.0))
}

// writeErrorMetric counts a failed consumption, identified by the type of error that failed it.
func (s *SubscriberCallback) writeErrorMetric(ctx context.Context, spec *ModelSpecification, err error) {
	s.metric.WriteOne(ctx, eventDatum(spec.ModelId, OutcomeApplied, metric.ErrorType(err), 1.0))
}

// eventDatum counts one model event a subscriber received. What became of it is carried by the outcome
// and by error.type, so a skip and a failure need no metric of their own.
func eventDatum(modelId string, outcome string, errorType string, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: MetricNameEvents,
		Dimensions: map[string]string{
			metric.DimensionModelId:   modelId,
			dimensionOutcome:          outcome,
			metric.DimensionErrorType: errorType,
		},
		Unit:  metric.UnitCount,
		Value: value,
		Kind:  metric.KindCounter.Build(),
	}
}

func getSubscriberCallbackDefaultMetrics(modelIds []string) []*metric.Datum {
	defaults := make([]*metric.Datum, 0, len(modelIds)*2)

	for _, modelId := range modelIds {
		defaults = append(defaults,
			eventDatum(modelId, OutcomeApplied, metric.DimensionDefault, 0.0),
			eventDatum(modelId, OutcomeSkipped, metric.DimensionDefault, 0.0),
		)
	}

	return defaults
}
