package mdlsub

import (
	"context"
	"fmt"
	"sort"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

//go:generate mockery --name SubscriberCore
type SubscriberCore interface {
	GetModelIds() []string
	GetLatestModelIdVersion(modelId mdl.ModelId) (int, error)
	GetTransformer(spec *ModelSpecification) (ModelTransformer, error)
	GetOutput(spec *ModelSpecification) (Output, error)
	Persist(ctx context.Context, spec *ModelSpecification, model Model) error
	Transform(ctx context.Context, spec *ModelSpecification, input any) (Model, error)
}

func NewSubscriberCore(ctx context.Context, config cfg.Config, logger log.Logger, subscriberSettings map[string]*SubscriberSettings, transformerFactories TransformerMapTypeVersionFactories) (SubscriberCore, error) {
	var err error
	var transformers ModelTransformers
	var outputs Outputs

	if transformers, err = initTransformers(ctx, config, logger, subscriberSettings, transformerFactories); err != nil {
		return nil, fmt.Errorf("failed to init transformers: %w", err)
	}

	if outputs, err = initOutputs(ctx, config, logger, subscriberSettings, transformers); err != nil {
		return nil, fmt.Errorf("failed to init outputs: %w", err)
	}

	return NewSubscriberCoreWithInterfaces(transformers, outputs), nil
}

func NewSubscriberCoreWithInterfaces(transformers ModelTransformers, outputs Outputs) SubscriberCore {
	return &subscriberCore{
		transformers: transformers,
		outputs:      outputs,
	}
}

type subscriberCore struct {
	transformers ModelTransformers
	outputs      Outputs
}

func (c *subscriberCore) GetModelIds() []string {
	return funk.Keys(c.transformers)
}

func (c *subscriberCore) GetLatestModelIdVersion(modelId mdl.ModelId) (int, error) {
	var ok bool
	var versioned VersionedModelTransformers

	str := modelId.String()

	if versioned, ok = c.transformers[str]; !ok {
		return 0, fmt.Errorf("failed to find model transformer for model id %s", str)
	}

	versions := funk.Keys(versioned)

	if len(versions) == 0 {
		return 0, fmt.Errorf("there are no versions available for transformer model id %s", str)
	}

	sort.Ints(versions)
	latest := versions[len(versions)-1]

	return latest, nil
}

func (c *subscriberCore) Transform(ctx context.Context, spec *ModelSpecification, input any) (Model, error) {
	var err error
	var model Model
	var transformer ModelTransformer

	if transformer, err = c.GetTransformer(spec); err != nil {
		return nil, fmt.Errorf("failed to get transformer: %w", err)
	}

	if model, err = transformer.Transform(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to transform %s: %w", spec, err)
	}

	return model, nil
}

func (c *subscriberCore) GetTransformer(spec *ModelSpecification) (ModelTransformer, error) {
	var ok bool

	if _, ok = c.transformers[spec.ModelId]; !ok {
		return nil, fmt.Errorf("there is no transformer for modelId %s", spec.ModelId)
	}

	if _, ok = c.transformers[spec.ModelId][spec.Version]; !ok {
		return nil, fmt.Errorf("there is no transformer for modelId %s and version %d", spec.ModelId, spec.Version)
	}

	return c.transformers[spec.ModelId][spec.Version], nil
}

func (c *subscriberCore) Persist(ctx context.Context, spec *ModelSpecification, model Model) error {
	var err error
	var output Output

	if output, err = c.GetOutput(spec); err != nil {
		return fmt.Errorf("failed to get output: %w", err)
	}

	if err = output.Persist(ctx, model, spec.CrudType); err != nil {
		return fmt.Errorf("failed to persist model %s: %w", spec, err)
	}

	return nil
}

func (c *subscriberCore) GetOutput(spec *ModelSpecification) (Output, error) {
	if _, ok := c.outputs[spec.ModelId]; !ok {
		return nil, fmt.Errorf("there is no output for modelId %s", spec.ModelId)
	}

	if _, ok := c.outputs[spec.ModelId][spec.Version]; !ok {
		return nil, fmt.Errorf("there is no output for modelId %s and version %d", spec.ModelId, spec.Version)
	}

	return c.outputs[spec.ModelId][spec.Version], nil
}
