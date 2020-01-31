package cli

import (
	"github.com/applike/gosoline/pkg/mon"
	"os"
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

	e.output.crossed = false
	e.output.Flush()

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

	return logger, nil
}
