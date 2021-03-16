package cli

import (
	"github.com/applike/gosoline/pkg/mon"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
)

type fingersCrossedOutput struct {
	size    int
	buffer  [][]byte
	crossed bool
}

func newFingersCrossedOutput(size int) *fingersCrossedOutput {
	return &fingersCrossedOutput{
		size:    size,
		buffer:  make([][]byte, 0, size),
		crossed: true,
	}
}

func (r *fingersCrossedOutput) Flush() {
	for i := 0; i < len(r.buffer); i++ {
		_, _ = os.Stdout.Write(r.buffer[i])
	}

	r.buffer = make([][]byte, 0, r.size)
}

func (r *fingersCrossedOutput) Write(line []byte) (n int, err error) {
	if !r.crossed {
		return os.Stdout.Write(line)
	}

	r.buffer = append(r.buffer, line)

	if len(r.buffer) > r.size {
		r.buffer = r.buffer[128:]
	}

	return len(line), nil
}

func (r *fingersCrossedOutput) enableLogs() {
	r.crossed = false
	r.Flush()
}

type errorHook struct {
	output *fingersCrossedOutput
}

func newErrorHook(output *fingersCrossedOutput) *errorHook {
	return &errorHook{
		output: output,
	}
}

func (e *errorHook) Fire(_ string, _ string, err error, _ *mon.Metadata) error {
	if err == nil {
		return nil
	}

	e.output.enableLogs()

	return nil
}

func newCliLogger() (mon.Logger, error) {
	output := newFingersCrossedOutput(1024)
	hook := newErrorHook(output)

	logger := mon.NewLogger()
	options := []mon.LoggerOption{
		mon.WithOutput(output),
		mon.WithHook(hook),
	}

	if err := logger.Option(options...); err != nil {
		return nil, err
	}

	go func() {
		// if we are using the status.StatusManager with the cli app, we have to enable the output upon receiving
		// a signal. Otherwise our logs will be swallowed and the status manager can't do its work.

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, unix.SIGUSR1)

		for range sigChan {
			output.enableLogs()
			logger.Info("handling USR1 signal; enabled output")
		}
	}()

	return logger, nil
}
