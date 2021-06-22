package kinesis_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	configMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud/aws/kinesis"
	kinesisMocks "github.com/applike/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/log"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func mockFactory(kinsumerMock kinesis.Kinsumer) kinesis.KinsumerFactory {
	return func(config cfg.Config, logger log.Logger, settings kinesis.KinsumerSettings) (kinesis.Kinsumer, error) {
		return kinsumerMock, nil
	}
}

func TestReaderLifeCycle(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := new(logMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)
	loggerMock.On("WithContext", mock.Anything).Return(loggerMock)

	msg := stream.NewMessage("foobar", map[string]interface{}{
		"bla": "blub",
	})
	bytes, _ := json.Marshal(msg)

	kinsumerMock := new(kinesisMocks.Kinsumer)
	kinsumerMock.On("Run").Return(nil).Once()
	kinsumerMock.On("Next").Return(bytes, nil).Once()
	kinsumerMock.On("Next").Return(nil, nil).Once()
	kinsumerMock.On("Stop").Once()

	factory := mockFactory(kinsumerMock)

	reader, err := stream.NewKinesisInput(configMock, loggerMock, factory, kinesis.KinsumerSettings{})
	assert.NoError(t, err)

	go func() {
		err := reader.Run(context.Background())
		assert.NoError(t, err)
	}()

	out, ok := <-reader.Data()
	assert.True(t, ok)
	assert.Equal(t, msg, out, "the messages should match")

	reader.Stop()

	kinsumerMock.AssertExpectations(t)
}

func TestReaderRunErrorInClientRun(t *testing.T) {
	configMock := new(configMocks.Config)
	loggerMock := new(logMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)
	loggerMock.On("WithContext", mock.Anything).Return(loggerMock)

	kinsumerMock := new(kinesisMocks.Kinsumer)
	kinsumerMock.On("Run").Return(errors.New("error")).Once()

	factory := mockFactory(kinsumerMock)

	reader, err := stream.NewKinesisInput(configMock, loggerMock, factory, kinesis.KinsumerSettings{})
	assert.NoError(t, err)

	err = reader.Run(context.Background())
	assert.Error(t, err)

	kinsumerMock.AssertExpectations(t)
}

func TestReaderRestartTrigger(t *testing.T) {
	configMock := new(configMocks.Config)

	loggerMock := new(logMocks.Logger)
	loggerMock.On("WithFields", mock.Anything).Return(loggerMock)
	loggerMock.On("WithContext", mock.Anything).Return(loggerMock)
	loggerMock.On("Info", mock.Anything)
	loggerMock.On("Warn", mock.Anything)

	msg := stream.NewMessage("foobar", map[string]interface{}{
		"bla": "blub",
	})
	bytes, err := json.Marshal(msg)
	assert.NoError(t, err)

	kinsumerMock := new(kinesisMocks.Kinsumer)
	kinsumerMock.On("Run").Return(nil).Twice()
	kinsumerMock.On("Next").Return(nil, fmt.Errorf("ExpiredIteratorException")).Once()
	kinsumerMock.On("Next").Return(bytes, nil).Once()
	kinsumerMock.On("Next").Return(nil, nil).Once()
	kinsumerMock.On("Stop")

	factory := mockFactory(kinsumerMock)

	reader, err := stream.NewKinesisInput(configMock, loggerMock, factory, kinesis.KinsumerSettings{})
	assert.NoError(t, err)

	go func() {
		err := reader.Run(context.Background())
		assert.NoError(t, err)
	}()

	out, ok := <-reader.Data()
	assert.True(t, ok)
	assert.Equal(t, msg, out, "the messages should match")

	reader.Stop()

	kinsumerMock.AssertExpectations(t)
}
