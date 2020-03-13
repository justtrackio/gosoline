package stream_test

import (
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestNewConfigurableMultiOutput(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "stream.output.pipeline.type").Return("multiple")
	config.On("UnmarshalKey", "stream.output.pipeline", mock.AnythingOfType("*stream.MultipleOutputConfiguration")).Run(func(args mock.Arguments) {
		cfg := args.Get(1).(*stream.MultipleOutputConfiguration)
		cfg.Outputs = []string{
			"outputA",
			"outputB",
		}
	})
	config.On("GetString", "stream.output.outputA.type").Return("file")
	config.On("UnmarshalKey", "stream.output.outputA", mock.AnythingOfType("*stream.FileOutputSettings")).Run(func(args mock.Arguments) {
		cfg := args.Get(1).(*stream.FileOutputSettings)
		cfg.Append = true
		cfg.Filename = "/tmp/temp-1"
	})
	config.On("GetString", "stream.output.outputB.type").Return("file")
	config.On("UnmarshalKey", "stream.output.outputB", mock.AnythingOfType("*stream.FileOutputSettings")).Run(func(args mock.Arguments) {
		cfg := args.Get(1).(*stream.FileOutputSettings)
		cfg.Append = true
		cfg.Filename = "/tmp/temp-2"
	})

	logger := mocks.NewLoggerMockedAll()

	_ = stream.NewConfigurableOutput(config, logger, "pipeline")

	config.AssertExpectations(t)
}
