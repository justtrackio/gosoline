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

func (m *multiOutput) ProvidesCompression() bool {
	for _, o := range m.outputs {
		if o.ProvidesCompression() {
			return false
		}
	}

	return true
}

func (m *multiOutput) SupportsAggregation() bool {
	for _, o := range m.outputs {
		if !o.SupportsAggregation() {
			return false
		}
	}

	return true
}

func (m *multiOutput) IsPartitionedOutput() bool {
	for _, o := range m.outputs {
		if po, ok := o.(PartitionedOutput); !ok || !po.IsPartitionedOutput() {
			return false
		}
	}

	return true
}

func (m *multiOutput) GetMaxMessageSize() (maxMessageSize *int) {
	for _, o := range m.outputs {
		if sro, ok := o.(SizeRestrictedOutput); ok {
			outputMaxMessageSize := sro.GetMaxMessageSize()
			if (maxMessageSize == nil && outputMaxMessageSize != nil) || (maxMessageSize != nil && outputMaxMessageSize != nil && *maxMessageSize > *outputMaxMessageSize) {
				maxMessageSize = outputMaxMessageSize
			}
		}
	}

	return
}

func (m *multiOutput) GetMaxBatchSize() (maxBatchSize *int) {
	for _, o := range m.outputs {
		if sro, ok := o.(SizeRestrictedOutput); ok {
			outputMaxBatchSize := sro.GetMaxBatchSize()
			if (maxBatchSize == nil && outputMaxBatchSize != nil) || (maxBatchSize != nil && outputMaxBatchSize != nil && *maxBatchSize > *outputMaxBatchSize) {
				maxBatchSize = outputMaxBatchSize
			}
		}
	}

	return
}

func NewConfigurableMultiOutput(ctx context.Context, config cfg.Config, logger log.Logger, base string) (Output, error) {
	key := fmt.Sprintf("%s.types", ConfigurableOutputKey(base))

	val, err := config.Get(key)
	if err != nil {
		return nil, fmt.Errorf("can not get output types: %w", err)
	}

	ts := val.(map[string]any)

	multiOutput := &multiOutput{
		outputs: make([]Output, 0),
	}

	for outputName := range ts {
		name := fmt.Sprintf("%s.types.%s", base, outputName)

		if output, err := NewConfigurableOutput(ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("can not create multi output %s: %w", base, err)
		} else {
			multiOutput.outputs = append(multiOutput.outputs, output)
		}
	}

	return multiOutput, nil
}
