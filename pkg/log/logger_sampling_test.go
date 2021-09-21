package log_test

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestSamplingLogger_Info(t *testing.T) {
	mock := new(mocks.Logger)
	mock.On("Info", "this should be logged").Once()
	mock.On("Info", "log msg", "a", 4).Twice()

	clock := clockwork.NewFakeClock()
	logger := log.NewSamplingLoggerWithInterfaces(mock, clock, time.Minute)

	logger.Info("log msg", "a", 4)
	logger.Info("log msg", "a", 4)
	logger.Info("this should be logged")

	clock.Advance(time.Second)
	logger.Info("log msg", "a", 4)

	clock.Advance(time.Hour)
	logger.Info("log msg", "a", 4)

	mock.AssertExpectations(t)
}
