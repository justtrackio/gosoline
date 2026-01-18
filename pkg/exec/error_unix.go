//go:build linux || darwin

package exec

import (
	"errors"

	"golang.org/x/sys/unix"
)

func isOsConnectionError(err error) bool {
	return errors.Is(err, unix.ECONNREFUSED) ||
		errors.Is(err, unix.ECONNRESET) ||
		errors.Is(err, unix.EPIPE)
}

func isOsTimeoutError(err error) bool {
	return errors.Is(err, unix.ETIMEDOUT)
}
