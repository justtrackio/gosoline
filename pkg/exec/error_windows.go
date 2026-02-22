//go:build windows

package exec

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isOsConnectionError(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED) ||
		errors.Is(err, windows.WSAECONNRESET)
}

func isOsTimeoutError(err error) bool {
	return errors.Is(err, windows.WSAETIMEDOUT)
}
