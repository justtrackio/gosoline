//go:build linux || darwin

package kernel

import (
	"os"

	"golang.org/x/sys/unix"
)

var interruptSignals = []os.Signal{unix.SIGINT, unix.SIGTERM}
