//go:build windows

package kernel

import (
	"os"
)

var interruptSignals = []os.Signal{os.Interrupt}
