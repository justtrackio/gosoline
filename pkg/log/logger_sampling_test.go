package log_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestSamplingLogger_Info(t *testing.T) {
	mock := mocks.NewLogger(t)
	mock.EXPECT().Info("this should be logged").Once()
	mock.EXPECT().Info("log msg", "a", 4).Twice()

	testClock := clock.NewFakeClock()
	logger := log.NewSamplingLoggerWithInterfaces(mock, testClock, time.Minute)

	logger.Info("log msg", "a", 4)
	logger.Info("log msg", "a", 4)
	logger.Info("this should be logged")

	testClock.Advance(time.Second)
	logger.Info("log msg", "a", 4)

	testClock.Advance(time.Hour)
	logger.Info("log msg", "a", 4)
}
