package db

import (
	"database/sql/driver"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewExecutor(config cfg.Config, logger log.Logger, name string, backoffType string, notifier ...exec.Notify) exec.Executor {
	res := &exec.ExecutableResource{
		Type: "db-client",
		Name: name,
	}

	executorSettings := exec.ReadBackoffSettings(config, backoffType)
	executor := exec.NewBackoffExecutor(logger, res, &executorSettings, []exec.ErrorChecker{
		exec.CheckConnectionError,
		exec.CheckTimeoutError,
		CheckInvalidConnection,
		CheckBadConnection,
		CheckIoTimeout,
	}, notifier...)

	return executor
}

func CheckInvalidConnection(result interface{}, err error) exec.ErrorType {
	if errors.Is(err, mysql.ErrInvalidConn) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckBadConnection(result interface{}, err error) exec.ErrorType {
	if errors.Is(err, driver.ErrBadConn) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckIoTimeout(result interface{}, err error) exec.ErrorType {
	if strings.Contains(err.Error(), "i/o timeout") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}
