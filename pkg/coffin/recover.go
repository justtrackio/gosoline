package coffin

import (
	"fmt"
	"runtime"
	"strings"
)

func ResolveRecovery(unknownErr any) error {
	switch rval := unknownErr.(type) {
	case nil:
		return nil

	case error:
		return withStack(rval)

	case string:
		return withStack(fmt.Errorf("%s", rval))

	default:
		return withStack(fmt.Errorf("unhandled error type %T", rval))
	}
}

func withStack(err error) error {
	const depth = 32
	var pcs [depth]uintptr

	n := runtime.Callers(3, pcs[:])
	st := runtime.CallersFrames(pcs[0:n])

	stack := make([]string, 0, n)
	done := false
	for !done {
		frame, more := st.Next()

		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))

		done = !more
	}

	return fmt.Errorf("%w\nstack:\n\t%s", err, strings.Join(stack, "\n\t"))
}
