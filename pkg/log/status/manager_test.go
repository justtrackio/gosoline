package status_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/log/status"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	m := status.NewManager()
	assert.IsType(t, m, status.ProvideManager())
	assert.NotSame(t, m, status.ProvideManager())

	logger := logMocks.NewLogger(t)

	h := m.StartWork("test", 3)
	logger.EXPECT().Info(matcher.Context, "Work item %s: step %d / %d (%.2f %%)", "test", 0, 3, 0.0).Once()
	m.PrintReport(t.Context(), logger)

	h.ReportProgress(1, 50)
	logger.EXPECT().Info(matcher.Context, "Work item %s: step %d / %d (%.2f %%)", "test", 1, 3, 50.0).Once()
	m.PrintReport(t.Context(), logger)

	h.ReportDone()
	logger.EXPECT().Info(matcher.Context, "Work item %s: done", "test").Once()
	m.PrintReport(t.Context(), logger)

	h = m.StartWork("next test", 1)
	logger.EXPECT().Info(matcher.Context, "Work item %s: step %d / %d (%.2f %%)", "next test", 0, 1, 0.0).Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: done", "test").Once()
	m.PrintReport(t.Context(), logger)

	h.ReportError(fmt.Errorf("out of ideas"))
	logger.EXPECT().Info(matcher.Context, "Work item %s: failed with error %s", "next test", "out of ideas").Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: done", "test").Once()
	m.PrintReport(t.Context(), logger)

	err := m.Monitor("monitored", func() error {
		return nil
	})()
	assert.NoError(t, err)
	err = m.MonitorWithContext("monitored error", func(ctx context.Context) error {
		assert.Equal(t, t.Context(), ctx)

		return fmt.Errorf("error: success")
	})(t.Context())
	assert.EqualError(t, err, "error: success")
	err = m.MonitorWithContext("monitored panic", func(ctx context.Context) error {
		assert.Equal(t, t.Context(), ctx)

		panic(fmt.Errorf("panic error"))
	})(t.Context())
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "panic error"))
	logger.EXPECT().Info(matcher.Context, "Work item %s: done", "monitored").Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: failed with error %s", "monitored error", "error: success").Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: failed with error %s", "monitored panic", err.Error()).Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: failed with error %s", "next test", "out of ideas").Once()
	logger.EXPECT().Info(matcher.Context, "Work item %s: done", "test").Once()
	m.PrintReport(t.Context(), logger)
}
