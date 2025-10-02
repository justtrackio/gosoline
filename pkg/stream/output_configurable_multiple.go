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

func NewConfigurableMultiOutput(ctx context.Context, config cfg.Config, logger log.Logger, base string) (Output, *OutputCapabilities, error) {
	key := fmt.Sprintf("%s.types", ConfigurableOutputKey(base))

	val, err := config.Get(key)
	if err != nil {
		return nil, nil, fmt.Errorf("can not get output types: %w", err)
	}

	outputs := val.(map[string]any)

	multiOutput := &multiOutput{
		outputs: make([]Output, 0),
	}

	outputCapabilities := &OutputCapabilities{
		IsPartitionedOutput:               true,
		ProvidesCompression:               true,
		SupportsAggregation:               true,
		MaxBatchSize:                      nil,
		MaxMessageSize:                    nil,
		IgnoreProducerDaemonBatchSettings: false,
	}

	for outputName := range outputs {
		name := fmt.Sprintf("%s.types.%s", base, outputName)

		componentOutput, componentCapabilities, err := NewConfigurableOutput(ctx, config, logger, name)
		if err != nil {
			return nil, nil, fmt.Errorf("can not create multi output %s: %w", base, err)
		}

		updateMultiOutputCapabilities(outputCapabilities, componentCapabilities)

		multiOutput.outputs = append(multiOutput.outputs, componentOutput)
	}

	return multiOutput, outputCapabilities, nil
}

func updateMultiOutputCapabilities(multiOutputCapabilities *OutputCapabilities, componentCapabilities *OutputCapabilities) {
	if (multiOutputCapabilities.MaxBatchSize == nil && componentCapabilities.MaxBatchSize != nil) ||
		(multiOutputCapabilities.MaxBatchSize != nil && componentCapabilities.MaxBatchSize != nil && *multiOutputCapabilities.MaxBatchSize > *componentCapabilities.MaxBatchSize) {
		multiOutputCapabilities.MaxBatchSize = componentCapabilities.MaxBatchSize
	}

	if (multiOutputCapabilities.MaxMessageSize == nil && componentCapabilities.MaxMessageSize != nil) ||
		(multiOutputCapabilities.MaxMessageSize != nil && componentCapabilities.MaxMessageSize != nil && *multiOutputCapabilities.MaxMessageSize > *componentCapabilities.MaxMessageSize) {
		multiOutputCapabilities.MaxMessageSize = componentCapabilities.MaxMessageSize
	}

	if !componentCapabilities.IsPartitionedOutput {
		multiOutputCapabilities.IsPartitionedOutput = false
	}

	if !componentCapabilities.ProvidesCompression {
		multiOutputCapabilities.ProvidesCompression = false
	}

	if !componentCapabilities.SupportsAggregation {
		multiOutputCapabilities.SupportsAggregation = false
	}

	if componentCapabilities.IgnoreProducerDaemonBatchSettings {
		multiOutputCapabilities.IgnoreProducerDaemonBatchSettings = true
	}
}
