package log_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
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

	s.JSONEq(`{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"a","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	s.JSONEq(`{"channel":"main","context":{},"fields":{},"level":3,"level_name":"warn","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
	s.JSONEq(`{"channel":"main","context":{},"err":"boom","fields":{},"level":4,"level_name":"error","message":"boom","timestamp":"1984-04-04T00:02:00Z"}`, lines[2])
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
	s.JSONEq(`{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"a","timestamp":"1984-04-04T00:00:00Z"}`, lines[0])
	s.JSONEq(`{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])

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
	s.JSONEq(`{"channel":"main","context":{},"fields":{},"level":2,"level_name":"info","message":"b","timestamp":"1984-04-04T00:01:00Z"}`, lines[1])
}

func (s *FingersCrossedSuite) TestWithoutScopeDropsNonError() {
	ctx := smplctx.WithSampling(s.T().Context(), smplctx.Sampling{Sampled: false})

	s.logger.Info(ctx, "a")
	s.Empty(s.buf.String())

	s.cl.Advance(time.Minute)
	s.logger.Error(ctx, "boom")

	lines := getLogLines(s.buf)
	s.Len(lines, 1)
	s.JSONEq(`{"channel":"main","context":{},"err":"boom","fields":{},"level":4,"level_name":"error","message":"boom","timestamp":"1984-04-04T00:01:00Z"}`, lines[0])
}

func TestFingersCrossedSuite(t *testing.T) {
	suite.Run(t, new(FingersCrossedSuite))
}
