package tidb

import (
	"strings"

	"github.com/justtrackio/gosoline/pkg/exec"
)

// Can be retried, but needs to recalculate business logic.
// Error 8002: can not retry select for update statement: SELECT FOR UPDATE write conflict error
func CheckSelectForUpdateWriteConflict(result any, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "Error 8002") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

// Can be retried but might need to account for recalculating business logic.
// Error 9007: Write conflict: Write conflict error, usually caused by multiple transactions modifying the same row of data when the optimistic transaction mode is used.
func CheckWriteConflict(result any, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "Error 9007") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

// Can be retried without accounting for business logic recalculation.
func CheckSchemaOutOfDate(result any, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "Error 8027") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

// Can be retried without accounting for business logic recalculation.
// Error 8028: Information schema is changed during the execution of the statement: Table schema has been changed by DDL operation, resulting in an error in the transaction commit.
func CheckSchemaChanged(result any, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "Error 8028") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

// Can be retried without accounting for business logic recalculation.
// Error 8022: Error: KV error safe to retry: transaction commit failed error.
func CheckTransactionError(result any, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "Error 8022") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}
