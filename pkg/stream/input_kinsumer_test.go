package stream_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/mon"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestReaderLifeCycle(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := new(monMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)

	msg := stream.Message{
		Attributes: map[string]interface{}{
			"bla": "blub",
		},
		Body: "foobar",
	}
	bytes, _ := json.Marshal(msg)

	kinsumerMock := new(streamMocks.Kinsumer)
	kinsumerMock.On("Run").Return(nil)
	kinsumerMock.On("Next").Return(bytes, nil).Once()
	kinsumerMock.On("Next").Return(nil, nil).Once()
	kinsumerMock.On("Stop")

	factory := func(config cfg.Config, logger mon.Logger, settings stream.KinsumerSettings) stream.Kinsumer {
		return kinsumerMock
	}

	var err error
	var out *stream.Message

	assert.NotPanics(t, func() {
		reader := stream.NewKinsumerInput(configMock, loggerMock, factory, stream.KinsumerSettings{})

		go func() {
			err = reader.Run(context.Background())
		}()

		out = <-reader.Data()

		reader.Stop()
	})

	assert.Nil(t, err, "there should be no error")
	assert.Equal(t, msg, *out, "the messages should match")
	kinsumerMock.AssertExpectations(t)
}

func TestReaderRunErrorInClientRun(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := new(monMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)

	kinsumerMock := new(streamMocks.Kinsumer)
	kinsumerMock.On("Run").Return(errors.New("error"))

	factory := func(config cfg.Config, logger mon.Logger, settings stream.KinsumerSettings) stream.Kinsumer {
		return kinsumerMock
	}

	assert.NotPanics(t, func() {
		reader := stream.NewKinsumerInput(configMock, loggerMock, factory, stream.KinsumerSettings{})

		assert.Panics(t, func() {
			_ = reader.Run(context.TODO())
		})
	})

	kinsumerMock.AssertExpectations(t)
}

func TestReaderRestartTrigger(t *testing.T) {
	configMock := new(configMocks.Config)

	loggerMock := new(monMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)
	loggerMock.On("Info", mock.Anything)
	loggerMock.On("Warn", mock.Anything)

	msg := stream.Message{
		Attributes: map[string]interface{}{
			"bla": "blub",
		},
		Body: "foobar",
	}
	bytes, _ := json.Marshal(msg)

	kinsumerMock := new(streamMocks.Kinsumer)
	kinsumerMock.On("Run").Return(nil)
	kinsumerMock.On("Next").Return(nil, fmt.Errorf("ExpiredIteratorException")).Once()
	kinsumerMock.On("Next").Return(bytes, nil).Once()
	kinsumerMock.On("Next").Return(nil, nil).Once()
	kinsumerMock.On("Stop")

	factory := func(config cfg.Config, logger mon.Logger, settings stream.KinsumerSettings) stream.Kinsumer {
		return kinsumerMock
	}

	var err error
	var out *stream.Message

	assert.NotPanics(t, func() {
		reader := stream.NewKinsumerInput(configMock, loggerMock, factory, stream.KinsumerSettings{})

		go func() {
			err = reader.Run(context.TODO())
		}()

		out = <-reader.Data()

		reader.Stop()
	})

	assert.Nil(t, err, "there should be no error")
	assert.Equal(t, msg, *out, "the messages should match")
	kinsumerMock.AssertExpectations(t)
}
