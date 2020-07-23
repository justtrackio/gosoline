package pubsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	ConfigKeyPubSubSubscribers = "pubsub.subscribers"
)

type SubscriberSettings struct {
	Input       string      `cfg:"input" default:"sns"`
	Output      string      `cfg:"output"`
	RunnerCount int         `cfg:"runner_count" default:"10" validate:"min=1"`
	SourceModel mdl.ModelId `cfg:"source"`
	TargetModel mdl.ModelId `cfg:"target"`
}

type SubscriberModel struct {
	cfg.AppId
	Name string `cfg:"name"`
}

type SubscriberCallback struct {
	logger       mon.Logger
	transformers ModelTransformers
	outputs      Outputs
}

func NewSubscriberCallback(transformers ModelTransformers, outputs Outputs) *SubscriberCallback {
	return &SubscriberCallback{
		transformers: transformers,
		outputs:      outputs,
	}
}

func (s *SubscriberCallback) Boot(config cfg.Config, logger mon.Logger) error {
	s.logger = logger
	return nil
}

func (s *SubscriberCallback) GetModel(attributes map[string]interface{}) interface{} {
	var err error
	var spec *ModelSpecification
	var transformer ModelTransformer

	if spec, err = getModelSpecification(attributes); err != nil {
		return nil
	}

	if transformer, err = s.getTransformer(spec); err != nil {
		return nil
	}

	return transformer.GetInput()
}

func (s *SubscriberCallback) Consume(ctx context.Context, input interface{}, attributes map[string]interface{}) (bool, error) {
	logger := s.logger.WithContext(ctx)

	var err error
	var model Model
	var spec *ModelSpecification
	var transformer ModelTransformer
	var output Output

	if spec, err = getModelSpecification(attributes); err != nil {
		return false, fmt.Errorf("can not read model specifications from the message attributes: %w", err)
	}

	if transformer, err = s.getTransformer(spec); err != nil {
		return false, err
	}

	if model, err = transformer.Transform(ctx, input); err != nil {
		return false, err
	}

	if model == nil {
		logger.Infof("skipping %s op for subscription for modelId %s and version %d", spec.CrudType, spec.ModelId, spec.Version)
		return true, nil
	}

	if output, err = s.getOutput(spec); err != nil {
		return false, err
	}

	if err = output.Persist(ctx, model, spec.CrudType); err != nil {
		return false, fmt.Errorf("can not persist subscription of model %s and version %d: %w", spec.ModelId, spec.Version, err)
	}

	logger.Infof("persisted %s op for subscription for modelId %s and version %d with id %v", spec.CrudType, spec.ModelId, spec.Version, model.GetId())

	return true, nil
}

func (s *SubscriberCallback) getTransformer(spec *ModelSpecification) (ModelTransformer, error) {
	var ok bool

	if _, ok = s.transformers[spec.ModelId]; !ok {
		return nil, fmt.Errorf("there is no transformer for modelId %s", spec.ModelId)
	}

	if _, ok = s.transformers[spec.ModelId][spec.Version]; !ok {
		return nil, fmt.Errorf("there is no transformer for modelId %s and version %d", spec.ModelId, spec.Version)
	}

	return s.transformers[spec.ModelId][spec.Version], nil
}

func (s *SubscriberCallback) getOutput(spec *ModelSpecification) (Output, error) {
	var ok bool

	if _, ok = s.transformers[spec.ModelId]; !ok {
		return nil, fmt.Errorf("there is no output for modelId %s", spec.ModelId)
	}

	if _, ok = s.transformers[spec.ModelId][spec.Version]; !ok {
		return nil, fmt.Errorf("there is no output for modelId %s and version %d", spec.ModelId, spec.Version)
	}

	return s.outputs[spec.ModelId][spec.Version], nil
}
