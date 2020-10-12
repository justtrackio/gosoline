package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type Output interface {
	Persist(ctx context.Context, model Model, op string) error
}

type Outputs map[string]map[int]Output
type OutputFactory func(config cfg.Config, logger mon.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) map[int]Output

var outputFactories = map[string]OutputFactory{}

func initOutputs(config cfg.Config, logger mon.Logger, subscriberSettings map[string]*SubscriberSettings, transformers ModelTransformers) (Outputs, error) {
	var ok bool
	var modelId string
	var outputs = make(Outputs)
	var outputFactory OutputFactory
	var versionedModelTransformers VersionedModelTransformers

	for name, settings := range subscriberSettings {
		modelId = settings.SourceModel.String()

		if outputFactory, ok = outputFactories[settings.Output]; !ok {
			return nil, fmt.Errorf("there is no output of type %s for subscriber %s with modelId %s", settings.Output, name, modelId)
		}

		if versionedModelTransformers, ok = transformers[modelId]; !ok {
			return nil, fmt.Errorf("there is no transformer for subscriber %s with modelId %s", name, modelId)
		}

		modelId := settings.SourceModel.String()
		outputs[modelId] = outputFactory(config, logger, settings, versionedModelTransformers)
	}

	return outputs, nil
}
