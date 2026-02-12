package errors

import (
	"errors"

	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sys/unix"
)

func IsRetryableKafkaError(err error) bool {
	switch {
	case exec.IsConnectionError(err): // Check for network-level connection errors (connection refused, reset, EOF, etc.)
		return true
	case kgo.IsRetryableBrokerErr(err): // Check if franz-go considers this a retryable broker error
		return true
	case kerr.IsRetriable(err): // Check if this is a retryable Kafka protocol error
		return true
	case exec.IsDNSNotFoundError(err): // Check for "no such host" errors. This might be temporary in some environments if a broker restarts.
		return true
	case errors.Is(err, unix.EHOSTUNREACH) || errors.Is(err, unix.ENETUNREACH): // Check for "no route to host" and "network unreachable" errors.
		return true
	default:
		return false
	}
}
