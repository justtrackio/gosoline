package exec

import (
	"errors"
	"io"
	"strings"

	"golang.org/x/sys/unix"
)

type ErrorType int

const (
	// We don't know yet, let the other error checkers decide about this error. If the error is
	// not marked retryable by another checker, we will not retry it.
	ErrorTypeUnknown ErrorType = iota
	// Stop retrying, the error was actually a "success" and needs to be propagated to the caller
	// ("success" meaning something e.g. was not found, but will not magically appear just because
	// we retry a few more times)
	ErrorTypeOk
	// Immediately stop retrying and return this error to the caller
	ErrorTypePermanent
	// Retry the execution of the action
	ErrorTypeRetryable
)

type ErrorChecker func(result interface{}, err error) ErrorType

func CheckUsedClosedConnectionError(_ interface{}, err error) ErrorType {
	if IsUsedClosedConnectionError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsUsedClosedConnectionError(err error) bool {
	return strings.Contains(err.Error(), "use of closed network connection")
}

func CheckConnectionError(_ interface{}, err error) ErrorType {
	if IsConnectionError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsConnectionError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, unix.ECONNREFUSED) || errors.Is(err, unix.ECONNRESET) || errors.Is(err, unix.EPIPE) {
		return true
	}

	if strings.Contains(err.Error(), "read: connection reset") {
		return true
	}

	return false
}

func CheckTimeoutError(_ interface{}, err error) ErrorType {
	if IsTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsTimeoutError(err error) bool {
	return errors.Is(err, unix.ETIMEDOUT)
}

func CheckClientAwaitHeaderTimeoutError(_ interface{}, err error) ErrorType {
	if IsClientAwaitHeadersTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsClientAwaitHeadersTimeoutError(err error) bool {
	return strings.Contains(err.Error(), "(Client.Timeout exceeded while awaiting headers)")
}

func CheckTlsHandshakeTimeoutError(_ interface{}, err error) ErrorType {
	if IsTlsHandshakeTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsTlsHandshakeTimeoutError(err error) bool {
	return strings.Contains(err.Error(), "net/http: TLS handshake timeout")
}
