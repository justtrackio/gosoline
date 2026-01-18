package exec

import (
	"errors"
	"io"
	"net"
	"strings"
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

type ErrorChecker func(result any, err error) ErrorType

func CheckUsedClosedConnectionError(_ any, err error) ErrorType {
	if IsUsedClosedConnectionError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsUsedClosedConnectionError(err error) bool {
	return strings.Contains(err.Error(), net.ErrClosed.Error())
}

func CheckConnectionError(_ any, err error) ErrorType {
	if IsConnectionError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsConnectionError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || isOsConnectionError(err) {
		return true
	}

	if err == nil {
		return false
	}

	if strings.Contains(err.Error(), "read: connection reset") {
		return true
	}

	return false
}

func CheckTimeoutError(_ any, err error) ErrorType {
	if IsTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsTimeoutError(err error) bool {
	return isOsTimeoutError(err)
}

func CheckClientAwaitHeaderTimeoutError(_ any, err error) ErrorType {
	if IsClientAwaitHeadersTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsClientAwaitHeadersTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "(Client.Timeout exceeded while awaiting headers)")
}

func CheckTlsHandshakeTimeoutError(_ any, err error) ErrorType {
	if IsTlsHandshakeTimeoutError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsTlsHandshakeTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "net/http: TLS handshake timeout")
}

func IsIoTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "i/o timeout")
}

func CheckOperationWasCanceledError(_ any, err error) ErrorType {
	if IsOperationWasCanceledError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsOperationWasCanceledError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "operation was canceled")
}

func CheckHttp2ClientConnectionForceClosedError(_ any, err error) ErrorType {
	if IsHttp2ClientConnectionForceClosedError(err) {
		return ErrorTypeRetryable
	}

	return ErrorTypeUnknown
}

func IsHttp2ClientConnectionForceClosedError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "http2: client connection force closed")
}
