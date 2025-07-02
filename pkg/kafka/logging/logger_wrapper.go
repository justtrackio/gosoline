package logging

import (
	"fmt"
	"io"
	"strings"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
)

// various errors related to the kafka clients metadata being outdated or some reorganization being in progress.
// these are not critical unless the max retry attempts are exceeded and in that case
// the library will actually return the error instead of logging it via the provided logger.
var nonCriticalKafkaErrors = []string{
	"Group Load In Progress",
	"Group Coordinator Not Available",
	"Not Leader For Partition",
	"not the leader",
	"Not Coordinator For Group",
	"Rebalance In Progress",
}

func isNonCriticalKafkaError(msg string) bool {
	return funk.ContainsFunc(nonCriticalKafkaErrors, func(nonCriticalError string) bool {
		return strings.Contains(msg, nonCriticalError)
	})
}

type DebugLoggerWrapper KafkaLogger

func (logger DebugLoggerWrapper) Printf(msg string, args ...any) {
	logger.Debug(msg, args...)
}

type ErrorLoggerWrapper KafkaLogger

func (logger ErrorLoggerWrapper) Printf(format string, args ...any) {
	err := fmt.Errorf(format, args...)

	if isNonCriticalKafkaError(err.Error()) ||
		strings.Contains(err.Error(), io.EOF.Error()) ||
		strings.Contains(err.Error(), io.ErrUnexpectedEOF.Error()) ||
		strings.Contains(err.Error(), "connection refused") ||
		exec.IsConnectionError(err) ||
		exec.IsIoTimeoutError(err) ||
		exec.IsUsedClosedConnectionError(err) ||
		exec.IsOperationWasCanceledError(err) {
		logger.Info(format, args...)

		return
	}

	logger.Error(format, args...)
}
