package mdlsub

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// OutputTypeFanout composes the output types configured under mdlsub.subscribers.<name>.outputs.
const OutputTypeFanout = "fanout"

func init() {
	AddOutput(OutputTypeFanout, outputFanoutFactory)
}

func outputFanoutFactory(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	settings *SubscriberSettings,
	transformers VersionedModelTransformers,
	subscriberName string,
) (map[int]Output, error) {
	if settings == nil {
		return nil, fmt.Errorf("can not create %s output without subscriber settings", OutputTypeFanout)
	}

	configKey := fmt.Sprintf("%s.outputs", GetSubscriberConfigKey(subscriberName))
	outputNames, err := config.GetStringSlice(configKey)
	if err != nil {
		return nil, fmt.Errorf("can not get configured outputs for subscriber %s: %w", subscriberName, err)
	}

	outputsByVersion := make(map[int][]fanoutOutputChild, len(transformers))
	for version := range transformers {
		outputsByVersion[version] = make([]fanoutOutputChild, 0, len(outputNames))
	}

	for _, outputName := range outputNames {
		if outputName == OutputTypeFanout {
			return nil, fmt.Errorf("can not configure %s output as a child of itself for subscriber %s", OutputTypeFanout, subscriberName)
		}

		factory, ok := outputFactories[outputName]
		if !ok {
			return nil, fmt.Errorf("there is no output of type %s configured for subscriber %s", outputName, subscriberName)
		}

		childOutputs, err := factory(ctx, config, logger, settings, transformers, subscriberName)
		if err != nil {
			return nil, fmt.Errorf("can not create child output %s for subscriber %s: %w", outputName, subscriberName, err)
		}

		for version := range transformers {
			childOutput, ok := childOutputs[version]
			if !ok {
				return nil, fmt.Errorf("child output %s has no output for version %d of subscriber %s", outputName, version, subscriberName)
			}
			if childOutput == nil {
				return nil, fmt.Errorf("child output %s returned a nil output for version %d of subscriber %s", outputName, version, subscriberName)
			}

			outputsByVersion[version] = append(outputsByVersion[version], fanoutOutputChild{
				name:   outputName,
				output: childOutput,
			})
		}
	}

	outputs := make(map[int]Output, len(outputsByVersion))
	for version, children := range outputsByVersion {
		outputs[version] = outputFanout{
			outputs: children,
		}
	}

	return outputs, nil
}

type fanoutOutputChild struct {
	name   string
	output Output
}

type outputFanout struct {
	outputs []fanoutOutputChild
}

// Persist deliberately invokes every child, even after one fails. Every child output must be idempotent.
func (o outputFanout) Persist(ctx context.Context, model Model, op string) error {
	errors := &multierror.Error{}

	for _, child := range o.outputs {
		if err := child.output.Persist(ctx, model, op); err != nil {
			errors = multierror.Append(errors, fmt.Errorf("output %s: %w", child.name, err))
		}
	}

	return errors.ErrorOrNil()
}
