package status_test

import (
	"context"
	"fmt"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/log/status"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestManager(t *testing.T) {
	m := status.NewManager()
	assert.IsType(t, m, status.ProvideManager())
	assert.NotSame(t, m, status.ProvideManager())

	logger := new(logMocks.Logger)

	h := m.StartWork("test", 3)
	logger.On("Info", "Work item %s: step %d / %d (%.2f %%)", "test", 0, 3, 0.0).Once()
	m.PrintReport(logger)

	h.ReportProgress(1, 50)
	logger.On("Info", "Work item %s: step %d / %d (%.2f %%)", "test", 1, 3, 50.0).Once()
	m.PrintReport(logger)

	h.ReportDone()
	logger.On("Info", "Work item %s: done", "test").Once()
	m.PrintReport(logger)

	h = m.StartWork("next test", 1)
	logger.On("Info", "Work item %s: step %d / %d (%.2f %%)", "next test", 0, 1, 0.0).Once()
	logger.On("Info", "Work item %s: done", "test").Once()
	m.PrintReport(logger)

	h.ReportError(fmt.Errorf("out of ideas"))
	logger.On("Info", "Work item %s: failed with error %s", "next test", "out of ideas").Once()
	logger.On("Info", "Work item %s: done", "test").Once()
	m.PrintReport(logger)

	err := m.Monitor("monitored", func() error {
		return nil
	})()
	assert.NoError(t, err)
	err = m.MonitorWithContext("monitored error", func(ctx context.Context) error {
		assert.Equal(t, context.Background(), ctx)

		return fmt.Errorf("error: success")
	})(context.Background())
	assert.EqualError(t, err, "error: success")
	err = m.MonitorWithContext("monitored panic", func(ctx context.Context) error {
		assert.Equal(t, context.Background(), ctx)

		panic(fmt.Errorf("panic error"))
	})(context.Background())
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "panic error"))
	logger.On("Info", "Work item %s: done", "monitored").Once()
	logger.On("Info", "Work item %s: failed with error %s", "monitored error", "error: success").Once()
	logger.On("Info", "Work item %s: failed with error %s", "monitored panic", err.Error()).Once()
	logger.On("Info", "Work item %s: failed with error %s", "next test", "out of ideas").Once()
	logger.On("Info", "Work item %s: done", "test").Once()
	m.PrintReport(logger)

	logger.AssertExpectations(t)
}
