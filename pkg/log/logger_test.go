package log_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestLoggerIoWriter(t *testing.T) {
	config := cfg.New(map[string]any{
		"log": map[string]any{
			"handlers": map[string]any{
				"main": map[string]any{
					"channels": map[string]any{
						"main": map[string]any{
							"level": log.LevelInfo,
						},
						"error_channel": map[string]any{
							"level": log.LevelError,
						},
						"forbidden": map[string]any{
							"level": log.LevelNone,
						},
					},
				},
			},
		},
	})

	buf := &bytes.Buffer{}
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, buf)
	cl := clock.NewFakeClock()

	logger := log.NewLoggerWithInterfaces(cl, []log.Handler{handler})

	ctx := t.Context()
	logger.Info(ctx, "foo")

	cl.Advance(time.Minute)
	logger.Info(ctx, "bar")
	logger.Debug(ctx, "some debug")
	logger.WithChannel("forbidden").Info(ctx, "something in forbidden channel")
	logger.WithChannel("error_channel").Warn(ctx, "not logged")
	logger.WithChannel("error_channel").Error(ctx, "error logged")

	cl.Advance(time.Minute)
	logger.Info(ctx, "foobaz")

	cl.Advance(time.Minute)
	err := fmt.Errorf("random error")
	logger.Error(t.Context(), "something went wrong: %w", err)

	lines := getLogLines(buf)
	assert.Len(t, lines, 5)

	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"foo","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"bar","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
	assert.JSONEq(t, `{"channel":"error_channel","context":{},"err":"error logged","fields":{},"level":4,"level_name":"error","message":"error logged","timestamp":"1984-04-04T00:01:00Z"}`, lines[2])
	assert.JSONEq(t, `{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"foobaz","timestamp":"1984-04-04T00:02:00Z"}`, lines[3])
	assert.JSONEq(t, `{"channel":"main","context":{},"err":"something went wrong: random error","fields":{},"level":4,"level_name":"error","message":"something went wrong: random error","timestamp":"1984-04-04T00:03:00Z"}`, lines[4])
}

func TestConfigureLoggerIoChannels(t *testing.T) {
	buf := &bytes.Buffer{}

	log.AddHandlerIoWriterFactory("buffer", func(config cfg.Config, configKey string) (io.Writer, error) {
		return buf, nil
	})

	t.Setenv("LOG_HANDLERS_MAIN_CHANNELS_VERBOSE_LEVEL", log.LevelNone)
	t.Setenv("LOG_HANDLERS_MAIN_CHANNELS_DEBUG_LEVEL", log.LevelDebug)

	config := cfg.New(map[string]any{
		"log": map[string]any{
			"handlers": map[string]any{
				"main": map[string]any{
					"type":   "iowriter",
					"writer": "buffer",
					"level":  log.LevelInfo,
					"channels": map[string]any{
						"debug": map[string]any{
							"level": log.LevelInfo,
						},
					},
				},
			},
		},
	})

	err := config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	assert.NoError(t, err)

	handlers, err := log.NewHandlersFromConfig(config)
	assert.NoError(t, err)

	logger := log.NewLogger()
	err = logger.Option(log.WithHandlers(handlers...))
	assert.NoError(t, err)

	ctx := t.Context()

	logger.WithChannel("default").Info(ctx, "should be logged")
	logger.WithChannel("debug").Debug(ctx, "should also be logged")
	logger.WithChannel("verbose").Warn(ctx, "should not be logged")
	logger.WithChannel("some-other-channel").Debug(ctx, "should not be logged")

	lines := getLogLines(buf)

	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "should be logged")
	assert.Contains(t, lines[1], "should also be logged")
}

func TestLoggerContextFields(t *testing.T) {
	config := cfg.New(map[string]any{})
	logger, buf := getBufferedLogger(t, config)

	ctx := t.Context()
	logger.Info(ctx, "foo")

	ctx = log.AppendContextFields(ctx, map[string]any{"foo": "bar"})
	logger.Warn(ctx, "foo with context")

	lines := getLogLines(buf)
	assert.Len(t, lines, 2, "number of logged lines do not match")

	assert.JSONEq(t, `{"channel":"main","level":2,"level_name":"info","timestamp":"1984-04-04T00:00:00Z","message":"foo","fields":{},"context":{}}`, lines[0])
	assert.JSONEq(t, `{"channel":"main","level":3,"level_name":"warn","timestamp":"1984-04-04T00:00:00Z","message":"foo with context","fields":{},"context":{"foo":"bar"}}`, lines[1])

	fmt.Println(lines[1])
}

func getBufferedLogger(t *testing.T, config cfg.Config) (log.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, buf)
	cl := clock.NewFakeClock()

	logger := log.NewLoggerWithInterfaces(cl, []log.Handler{handler})
	if err := logger.Option(log.WithContextFieldsResolver(log.ContextFieldsResolver)); err != nil {
		t.Fatal(err)
	}

	return logger, buf
}

func getLogLines(buf *bytes.Buffer) []string {
	return funk.Filter(strings.Split(buf.String(), "\n"), func(s string) bool {
		return s != ""
	})
}
