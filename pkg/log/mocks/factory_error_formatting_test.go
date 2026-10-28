package mocks

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLoggerErrorDiagnosticFormatsWrappedError(t *testing.T) {
	logger := NewLoggerMock(WithTestingT(t), WithMockAll)
	sentinel := errors.New("retry queue unavailable")
	cause := fmt.Errorf("outer: %w", sentinel)

	logger.Error(t.Context(), "can not put the message into the retry handler: %w", cause)

	logs := logger.(*loggerMock).pendingLogs[t.Name()]
	if len(logs) != 1 {
		t.Fatalf("expected one captured log, got %d", len(logs))
	}

	got := logs[0].message
	if !strings.Contains(got, "can not put the message into the retry handler") {
		t.Fatalf("formatted error log lost context: %q", got)
	}

	if !strings.Contains(got, "outer: retry queue unavailable") {
		t.Fatalf("formatted error log lost wrapped cause: %q", got)
	}

	if strings.Contains(got, "%!w") {
		t.Fatalf("formatted error log contains a formatting diagnostic: %q", got)
	}
}
