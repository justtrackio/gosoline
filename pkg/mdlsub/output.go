package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Output interface {
	Persist(ctx context.Context, model Model, op string) error
}

type (
	Outputs       map[string]map[int]Output
	OutputFactory func(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error)
)

var outputFactories = map[string]OutputFactory{}

func initOutputs(ctx context.Context, config cfg.Config, logger log.Logger, subscriberSettings map[string]*SubscriberSettings, transformers ModelTransformers) (Outputs, error) {
	var ok bool
	var err error
	var modelId string
	outputs := make(Outputs)
	var outputFactory OutputFactory
	var versionedModelTransformers VersionedModelTransformers

	for name, settings := range subscriberSettings {
		modelId = settings.SourceModel.String()

		if outputFactory, ok = outputFactories[settings.Output]; !ok {
			return nil, fmt.Errorf("can not create outputs: there is no output of type %s for subscriber %s with modelId %s", settings.Output, name, modelId)
		}

		if versionedModelTransformers, ok = transformers[modelId]; !ok {
			return nil, fmt.Errorf("can not create transformer: there is no transformer for subscriber %s with modelId %s", name, modelId)
		}

		modelId := settings.SourceModel.String()

		if outputs[modelId], err = outputFactory(ctx, config, logger, settings, versionedModelTransformers); err != nil {
			return nil, fmt.Errorf("can not create output for subscriber %s with modelId %s: %w", name, modelId, err)
		}
	}

	return outputs, nil
}
