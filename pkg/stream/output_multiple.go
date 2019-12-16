package stream

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/hashicorp/go-multierror"
)

type multiOutput struct {
	Outputs []Output
}

func NewConfigurableMultiOutput(_ cfg.Config, _ mon.Logger, outputs []Output) Output {
	return &multiOutput{
		Outputs: outputs,
	}
}

func NewConfigurableMultiOutputWithInterfaces(_ mon.Logger, outputs []Output) Output {
	return &multiOutput{
		Outputs: outputs,
	}
}

func (m *multiOutput) WriteOne(ctx context.Context, msg *Message) error {
	err := &multierror.Error{}

	for _, output := range m.Outputs {
		err = multierror.Append(err, output.WriteOne(ctx, msg))
	}

	return err.ErrorOrNil()
}

func (m *multiOutput) Write(ctx context.Context, batch []*Message) error {
	err := &multierror.Error{}

	for _, output := range m.Outputs {
		err = multierror.Append(err, output.Write(ctx, batch))
	}

	return err.ErrorOrNil()
}
