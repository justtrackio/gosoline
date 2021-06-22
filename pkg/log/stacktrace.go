package log

import (
	"runtime"
	"strconv"
	"strings"
)

type StackTraceProvider func(depthSkip int) string

func GetMockedStackTrace(depthSkip int) string {
	return "mocked trace"
}

// GetStackTrace constructs the current stacktrace. depthSkip defines how many steps of the
// stacktrace should be skipped. This is useful to not clutter the stacktrace with logging
// function calls.
func GetStackTrace(depthSkip int) string {
	depthSkip = depthSkip + 1 // Skip this function in stacktrace
	maxDepth := 50
	traces := make([]string, 0, maxDepth)

	// Get traces
	var depth int
	for depth = 0; depth < maxDepth; depth++ {
		function, _, line, ok := runtime.Caller(depth)

		if !ok {
			break
		}

		var traceStrBuilder strings.Builder
		traceStrBuilder.WriteString("\t")
		traceStrBuilder.WriteString(runtime.FuncForPC(function).Name())
		traceStrBuilder.WriteString(":")
		traceStrBuilder.WriteString(strconv.Itoa(line))
		traceStrBuilder.WriteString("\n")
		traces = append(traces, traceStrBuilder.String())
	}

	// Assemble stacktrace in reverse order
	var strBuilder strings.Builder
	strBuilder.WriteString("\n")

	for i := len(traces) - 1; i > depthSkip; i-- {
		strBuilder.WriteString(traces[i])
	}
	return strBuilder.String()
}
