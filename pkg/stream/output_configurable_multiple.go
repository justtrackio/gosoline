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

	outputs, err := config.GetStringMap(key)
	if err != nil {
		return nil, nil, fmt.Errorf("can not get output types: %w", err)
	}

	multiOutput := &multiOutput{
		outputs: make([]Output, 0),
	}

	outputCapabilities := &OutputCapabilities{
		IsPartitionedOutput:               false,
		ProvidesCompression:               false,
		SupportsAggregation:               true,
		MaxBatchSize:                      nil,
		MaxMessageSize:                    nil,
		IgnoreProducerDaemonBatchSettings: false,
	}

	atLeastOneWithoutPartitioningSupport := false

	for outputName := range outputs {
		name := fmt.Sprintf("%s.types.%s", base, outputName)

		componentOutput, componentCapabilities, err := NewConfigurableOutput(ctx, config, logger, name)
		if err != nil {
			return nil, nil, fmt.Errorf("can not create multi output %s: %w", base, err)
		}

		if !componentCapabilities.IsPartitionedOutput {
			atLeastOneWithoutPartitioningSupport = true
		}

		updateMultiOutputCapabilities(outputCapabilities, componentCapabilities, atLeastOneWithoutPartitioningSupport)

		multiOutput.outputs = append(multiOutput.outputs, componentOutput)
	}

	return multiOutput, outputCapabilities, nil
}

func updateMultiOutputCapabilities(multiOutputCapabilities *OutputCapabilities, componentCapabilities *OutputCapabilities, atLeastOneWithoutPartitioningSupport bool) {
	if componentCapabilities.MaxBatchSize != nil &&
		(multiOutputCapabilities.MaxBatchSize == nil || *multiOutputCapabilities.MaxBatchSize > *componentCapabilities.MaxBatchSize) {
		multiOutputCapabilities.MaxBatchSize = componentCapabilities.MaxBatchSize
	}

	if componentCapabilities.MaxMessageSize != nil &&
		(multiOutputCapabilities.MaxMessageSize == nil || *multiOutputCapabilities.MaxMessageSize > *componentCapabilities.MaxMessageSize) {
		multiOutputCapabilities.MaxMessageSize = componentCapabilities.MaxMessageSize
	}

	if atLeastOneWithoutPartitioningSupport {
		multiOutputCapabilities.IsPartitionedOutput = false
	} else {
		multiOutputCapabilities.IsPartitionedOutput = componentCapabilities.IsPartitionedOutput
	}

	if componentCapabilities.ProvidesCompression {
		multiOutputCapabilities.ProvidesCompression = true
	}

	if !componentCapabilities.SupportsAggregation {
		multiOutputCapabilities.SupportsAggregation = false
	}

	if componentCapabilities.IgnoreProducerDaemonBatchSettings {
		multiOutputCapabilities.IgnoreProducerDaemonBatchSettings = true
	}
}
