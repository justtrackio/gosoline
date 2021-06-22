package log_test

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func TestLoggerIoWriter(t *testing.T) {
	var buf = &bytes.Buffer{}
	var handler = log.NewHandlerIoWriter(log.LevelInfo, []string{"main"}, log.FormatterJson, time.RFC3339, buf)
	var cl = clock.NewFakeClock()

	logger := log.NewLoggerWithInterfaces(cl, []log.Handler{handler})

	logger.Info("foo")

	cl.Advance(time.Minute)
	logger.Info("bar")
	logger.Debug("some debug")
	logger.WithChannel("other channel").Info("something in another channel")

	cl.Advance(time.Minute)
	logger.Info("foobaz")

	cl.Advance(time.Minute)
	err := fmt.Errorf("random error")
	logger.Error("something went wrong: %w", err)

	lines := getLogLines(buf)
	assert.Len(t, lines, 4)

	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"foo","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"bar","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"foobaz","timestamp":"1984-04-04T00:02:00Z"}`, lines[2])
	assert.JSONEq(t, `{"channel":"main","context":{},"err":"something went wrong: random error","fields":{},"level":4,"level_name":"error","message":"something went wrong: random error","timestamp":"1984-04-04T00:03:00Z"}`, lines[3])
}

func getLogLines(buf *bytes.Buffer) []string {
	lines := make([]string, 0)

	for _, line := range strings.Split(buf.String(), "\n") {
		if len(line) == 0 {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}
