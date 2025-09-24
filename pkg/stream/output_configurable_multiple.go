package stream

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type multiOutput struct {
	outputs []Output
}

func (m *multiOutput) WriteOne(ctx context.Context, msg WritableMessage) error {
	err := &multierror.Error{}

	for _, output := range m.outputs {
		err = multierror.Append(err, output.WriteOne(ctx, msg))
	}

	return err.ErrorOrNil()
}

func (m *multiOutput) Write(ctx context.Context, batch []WritableMessage) error {
	err := &multierror.Error{}

	for _, output := range m.outputs {
		err = multierror.Append(err, output.Write(ctx, batch))
	}

	return err.ErrorOrNil()
}

func NewConfigurableMultiOutput(ctx context.Context, config cfg.Config, logger log.Logger, base string) (Output, *OutputSettings, error) {
	key := fmt.Sprintf("%s.types", ConfigurableOutputKey(base))

	val, err := config.Get(key)
	if err != nil {
		return nil, nil, fmt.Errorf("can not get output types: %w", err)
	}

	outputs := val.(map[string]any)

	multiOutput := &multiOutput{
		outputs: make([]Output, 0),
	}

	outputSettings := &OutputSettings{
		IsPartitionedOutput:               true,
		ProvidesCompression:               true,
		SupportsAggregation:               true,
		MaxBatchSize:                      nil,
		MaxMessageSize:                    nil,
		IgnoreProducerDaemonBatchSettings: false,
	}

	for outputName := range outputs {
		name := fmt.Sprintf("%s.types.%s", base, outputName)

		componentOutput, componentSettings, err := NewConfigurableOutput(ctx, config, logger, name)
		if err != nil {
			return nil, nil, fmt.Errorf("can not create multi output %s: %w", base, err)
		}

		updateMultiOutputSettings(outputSettings, componentSettings)

		multiOutput.outputs = append(multiOutput.outputs, componentOutput)
	}

	return multiOutput, outputSettings, nil
}

func updateMultiOutputSettings(multiOutputSettings *OutputSettings, componentSettings *OutputSettings) {
	if (multiOutputSettings.MaxBatchSize == nil && componentSettings.MaxBatchSize != nil) ||
		(multiOutputSettings.MaxBatchSize != nil && componentSettings.MaxBatchSize != nil && *multiOutputSettings.MaxBatchSize > *componentSettings.MaxBatchSize) {
		multiOutputSettings.MaxBatchSize = componentSettings.MaxBatchSize
	}

	if (multiOutputSettings.MaxMessageSize == nil && componentSettings.MaxMessageSize != nil) ||
		(multiOutputSettings.MaxMessageSize != nil && componentSettings.MaxMessageSize != nil && *multiOutputSettings.MaxMessageSize > *componentSettings.MaxMessageSize) {
		multiOutputSettings.MaxMessageSize = componentSettings.MaxMessageSize
	}

	if !componentSettings.IsPartitionedOutput {
		multiOutputSettings.IsPartitionedOutput = false
	}

	if !componentSettings.ProvidesCompression {
		multiOutputSettings.ProvidesCompression = false
	}

	if !componentSettings.SupportsAggregation {
		multiOutputSettings.SupportsAggregation = false
	}

	if componentSettings.IgnoreProducerDaemonBatchSettings {
		multiOutputSettings.IgnoreProducerDaemonBatchSettings = true
	}
}
