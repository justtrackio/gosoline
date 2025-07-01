package db

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewExecutor(config cfg.Config, logger log.Logger, name string, backoffType string, notifier ...exec.Notify) exec.Executor {
	return NewExecutorWithChecker(config, logger, name, backoffType, []exec.ErrorChecker{}, notifier)
}

func NewExecutorWithChecker(config cfg.Config, logger log.Logger, name string, backoffType string, checker []exec.ErrorChecker, notifier []exec.Notify) exec.Executor {
	res := &exec.ExecutableResource{
		Type: "db-client",
		Name: name,
	}

	executorSettings := exec.ReadBackoffSettings(config, backoffType)

	return exec.NewExecutor(
		logger,
		res,
		&executorSettings,
		append([]exec.ErrorChecker{
			exec.CheckConnectionError,
			exec.CheckTimeoutError,
			CheckDeadlock,
			CheckInvalidConnection,
			CheckBadConnection,
			CheckIoTimeout,
		}, checker...),
		notifier...,
	)
}

func ExecutorBackoffType(name string) string {
	return fmt.Sprintf("db.%s.retry", name)
}

func CheckDeadlock(result any, err error) exec.ErrorType {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1213 {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckInvalidConnection(result any, err error) exec.ErrorType {
	if errors.Is(err, mysql.ErrInvalidConn) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckBadConnection(result any, err error) exec.ErrorType {
	if errors.Is(err, driver.ErrBadConn) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func CheckIoTimeout(result any, err error) exec.ErrorType {
	if exec.IsIoTimeoutError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}
