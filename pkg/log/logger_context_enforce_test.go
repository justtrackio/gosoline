package log_test

import (
	"bytes"
	"testing"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/suite"
)

type ContextEnforcingLoggerTestSuite struct {
	suite.Suite

	clock  clock.Clock
	output *bytes.Buffer
	base   log.Logger
	logger *log.ContextEnforcingLogger
}

func (s *ContextEnforcingLoggerTestSuite) SetupTest() {
	s.clock = clock.NewFakeClock()
	s.output = &bytes.Buffer{}
	s.base = log.NewLoggerWithInterfaces(s.clock, []log.Handler{
		log.NewHandlerIoWriter(log.LevelInfo, log.Channels{}, log.FormatterConsole, "15:04:05.000", s.output),
	})

	s.logger = log.NewContextEnforcingLoggerWithInterfaces(s.base, log.GetMockedStackTrace, s.base)
	s.logger.Enable()
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithContext() {
	ctx := s.T().Context()
	logger := s.logger.WithContext(ctx)

	logger.Info("this is a info message")

	s.Equal("00:00:00.000 main    info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContext() {
	s.logger.Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger: mocked trace\n00:00:00.000 main    info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContextWithChannel() {
	s.logger.WithChannel("Channel").Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger: mocked trace\n00:00:00.000 Channel info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContextWithFields() {
	s.logger.WithFields(log.Fields{}).Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger: mocked trace\n00:00:00.000 main    info    this is a info message\n", s.output.String())
}

func TestContextEnforcingLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(ContextEnforcingLoggerTestSuite))
}
