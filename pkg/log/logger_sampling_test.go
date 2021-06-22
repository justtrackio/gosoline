package log_test

import (
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/jonboulle/clockwork"
	"testing"
	"time"
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
