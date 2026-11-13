package log_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/stretchr/testify/suite"
)

type FingersCrossedSuite struct {
	suite.Suite
	logger log.GosoLogger
	buf    *bytes.Buffer
	cl     clock.FakeClock
}

func (s *FingersCrossedSuite) SetupTest() {
	s.buf = &bytes.Buffer{}
	// use Trace level to ensure everything passes the handler filter
	handler := log.NewHandlerIoWriter(cfg.New(), log.PriorityTrace, log.FormatterJson, "main", time.RFC3339, s.buf)
	s.cl = clock.NewFakeClock()

	s.logger = log.NewLoggerWithInterfaces(s.cl, []log.Handler{handler})

	// enable sampling to trigger fingers crossed logic
	err := s.logger.Option(log.WithSamplingEnabled(true))
	s.NoError(err)
}

func (s *FingersCrossedSuite) TestFlushOnError() {
	s.Require().NoError(s.logger.Option(log.WithSamplingPassthroughLevel(log.LevelWarn)))

	// by default contexts are sampled, so we need to disable sampling to trigger fingers crossed
	ctx := smplctx.WithSampling(s.T().Context(), smplctx.Sampling{Sampled: false})
	ctx = log.WithFingersCrossedScope(ctx)

	// should be buffered
	s.logger.Info(ctx, "a")
	s.cl.Advance(time.Minute)

	// should be buffered
	s.logger.Warn(ctx, "b")

	// nothing written yet
	s.Empty(s.buf.String())

	s.cl.Advance(time.Minute)
	// trigger flush
	s.logger.Error(ctx, "boom")

	lines := getLogLines(s.buf)
	s.Len(lines, 3)

	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"fields":{},"level":2,"level_name":"info","message":"a","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"fields":{},"level":3,"level_name":"warn","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"err":"boom","fields":{},"level":4,"level_name":"error","message":"boom","timestamp":"1984-04-04T00:02:00Z"}`, lines[2])
}

func (s *FingersCrossedSuite) TestManualFlush() {
	ctx := smplctx.WithSampling(s.T().Context(), smplctx.Sampling{Sampled: false})
	ctx = log.WithFingersCrossedScope(ctx)

	s.logger.Info(ctx, "a")
	s.cl.Advance(time.Minute)
	s.logger.Info(ctx, "b")

	s.Empty(s.buf.String())

	log.FlushFingersCrossedScope(ctx)

	lines := getLogLines(s.buf)
	s.Len(lines, 2)
	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"fields":{},"level":2,"level_name":"info","message":"a","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"fields":{},"level":2,"level_name":"info","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])

	// ensure no-op on context without scope
	log.FlushFingersCrossedScope(s.T().Context())
	s.Len(getLogLines(s.buf), 2)
}

func (s *FingersCrossedSuite) TestAfterFlushWritesImmediately() {
	ctx := smplctx.WithSampling(s.T().Context(), smplctx.Sampling{Sampled: false})
	ctx = log.WithFingersCrossedScope(ctx)

	s.logger.Info(ctx, "a")
	s.Empty(s.buf.String())

	log.FlushFingersCrossedScope(ctx)
	s.Len(getLogLines(s.buf), 1)

	s.cl.Advance(time.Minute)
	// should write immediately now
	s.logger.Info(ctx, "b")

	lines := getLogLines(s.buf)
	s.Len(lines, 2)
	s.JSONEq(`{"channel":"main","context":{"fingers_crossed_flushed":true,"sampled":false},"fields":{},"level":2,"level_name":"info","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
}

func (s *FingersCrossedSuite) TestWithoutScopePassthroughLevel() {
	for _, tc := range []struct {
		name  string
		level string
		lines int
	}{
		{name: "default", lines: 1},
		{name: "error", level: log.LevelError, lines: 1},
		{name: "warn", level: log.LevelWarn, lines: 2},
	} {
		s.Run(tc.name, func() {
			s.SetupTest()
			config := cfg.New()
			if tc.level != "" {
				s.Require().NoError(config.Option(cfg.WithConfigSetting("sampling.log_passthrough_level", tc.level)))
			}

			settings, err := smpl.UnmarshalSettings(config)
			s.Require().NoError(err)
			if tc.level != "" {
				s.Require().Equal(tc.level, settings.LogPassthroughLevel)
				s.Require().NoError(s.logger.Option(log.WithSamplingPassthroughLevel(settings.LogPassthroughLevel)))
			} else {
				s.Equal(log.LevelError, settings.LogPassthroughLevel)
			}

			ctx := smplctx.WithSampling(s.T().Context(), smplctx.Sampling{Sampled: false})
			for _, logger := range []log.Logger{s.logger, s.logger.WithChannel("child").WithFields(log.Fields{"derived": true})} {
				s.buf.Reset()
				logger.Debug(ctx, "debug")
				logger.Info(ctx, "info")
				logger.Warn(ctx, "warning")
				s.Len(getLogLines(s.buf), tc.lines-1)

				logger.Error(ctx, "boom")
				lines := getLogLines(s.buf)
				s.Require().Len(lines, tc.lines)
				s.Contains(lines[tc.lines-1], `"message":"boom"`)
				s.Contains(lines[tc.lines-1], `"sampled":false`)
				if tc.level == log.LevelWarn {
					s.Contains(lines[0], `"message":"warning"`)
				}
			}
		})
	}
}

func (s *FingersCrossedSuite) TestInvalidPassthroughLevel() {
	for _, level := range []string{log.LevelInfo, log.LevelNone, "invalid"} {
		s.Run(level, func() {
			config := cfg.New()
			s.Require().NoError(config.Option(cfg.WithConfigSetting("sampling.log_passthrough_level", level)))
			settings, err := smpl.UnmarshalSettings(config)
			s.ErrorContains(err, "LogPassthroughLevel")
			s.Nil(settings)
			s.ErrorContains(s.logger.Option(log.WithSamplingPassthroughLevel(level)), "must be warn or error")
		})
	}
}

func TestFingersCrossedSuite(t *testing.T) {
	suite.Run(t, new(FingersCrossedSuite))
}
