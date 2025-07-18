package redis

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewExecutor(logger log.Logger, settings exec.BackoffSettings, name string) exec.Executor {
	return NewBackoffExecutor(logger, settings, name)
}

func NewBackoffExecutor(logger log.Logger, settings exec.BackoffSettings, name string) exec.Executor {
	executableResource := &exec.ExecutableResource{
		Type: "redis",
		Name: name,
	}

	checks := []exec.ErrorChecker{
		RetryableErrorChecker,
		OOMChecker,
		NilChecker,
	}

	return exec.NewBackoffExecutor(logger, executableResource, &settings, checks)
}

func NilChecker(_ any, err error) exec.ErrorType {
	if errors.Is(err, Nil) {
		return exec.ErrorTypeOk
	}

	return exec.ErrorTypeUnknown
}

func OOMChecker(_ any, err error) exec.ErrorType {
	if strings.HasPrefix(err.Error(), "OOM") {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func RetryableErrorChecker(_ any, err error) exec.ErrorType {
	if IsRetryableError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if err == io.EOF {
		return true
	}

	if _, ok := err.(net.Error); ok {
		return true
	}

	s := err.Error()

	if s == "ERR max number of clients reached" {
		return true
	}

	if strings.HasPrefix(s, "LOADING ") {
		return true
	}

	if strings.HasPrefix(s, "READONLY ") {
		return true
	}

	if strings.HasPrefix(s, "CLUSTERDOWN ") {
		return true
	}

	return false
}
