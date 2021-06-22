package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	metricMocks "github.com/applike/gosoline/pkg/metric/mocks"
	redisMocks "github.com/applike/gosoline/pkg/redis/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestRedisListOutput_WriteOne(t *testing.T) {
	output, redisMock := setup(1)
	redisMock.On("RPush", mock.AnythingOfType("*context.emptyCtx"), "mcoins-test-analytics-app-my-list", mock.AnythingOfType("[]uint8")).Return(int64(1), nil).Once()

	record := stream.NewMessage("bla")
	err := output.WriteOne(context.Background(), record)

	assert.Nil(t, err, "there should be no error")
	redisMock.AssertExpectations(t)
}

func TestRedisListOutput_Write(t *testing.T) {
	output, redisMock := setup(2)
	redisMock.On("RPush", mock.AnythingOfType("*context.emptyCtx"), "mcoins-test-analytics-app-my-list", mock.AnythingOfType("[]uint8"), mock.AnythingOfType("[]uint8")).Return(int64(2), nil).Once()

	batch := []stream.WritableMessage{
		stream.NewMessage("foo"),
		stream.NewMessage("bar"),
	}
	err := output.Write(context.Background(), batch)

	assert.Nil(t, err, "there should be no error")
	redisMock.AssertExpectations(t)
}

func TestRedisListOutput_Write_Chunked(t *testing.T) {
	output, redisMock := setup(1)
	redisMock.On("RPush", mock.AnythingOfType("*context.emptyCtx"), "mcoins-test-analytics-app-my-list", mock.AnythingOfType("[]uint8")).Return(int64(1), nil).Times(2)

	batch := []stream.WritableMessage{
		stream.NewMessage("foo"),
		stream.NewMessage("bar"),
	}
	err := output.Write(context.Background(), batch)

	assert.Nil(t, err, "there should be no error")
	redisMock.AssertExpectations(t)
}

func setup(batchSize int) (stream.Output, *redisMocks.Client) {
	loggerMock := logMocks.NewLoggerMockedAll()
	mw := metricMocks.NewWriterMockedAll()

	redisMock := new(redisMocks.Client)
	output := stream.NewRedisListOutputWithInterfaces(loggerMock, mw, redisMock, getSettings(batchSize))

	return output, redisMock
}

func getSettings(batchSize int) *stream.RedisListOutputSettings {
	return &stream.RedisListOutputSettings{
		AppId: cfg.AppId{
			Project:     "mcoins",
			Environment: "test",
			Family:      "analytics",
			Application: "app",
		},
		Key:       "my-list",
		BatchSize: batchSize,
	}
}
