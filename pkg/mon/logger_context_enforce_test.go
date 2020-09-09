package mon_test

import (
	"bytes"
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ContextEnforcingLoggerTestSuite struct {
	suite.Suite

	clock  clockwork.Clock
	output *bytes.Buffer
	base   mon.GosoLog
	logger *mon.ContextEnforcingLogger
}

func (s *ContextEnforcingLoggerTestSuite) SetupTest() {
	s.clock = clockwork.NewFakeClock()
	s.output = &bytes.Buffer{}
	logger := mon.NewLoggerWithInterfaces()

	handler, err := mon.NewIowriterLoggerHandler(s.clock, mon.FormatConsole, s.output, "15:04:05.000", []string{mon.Info, mon.Warn})
	require.Nil(s.T(), err)

	opt := mon.WithHandler(handler)
	err = opt(logger)
	require.Nil(s.T(), err)

	s.base = logger

	s.logger = mon.NewContextEnforcingLoggerWithInterfaces(s.base, mon.GetMockedStackTrace, s.base)
	s.logger.Enable()
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithContext() {
	ctx := context.Background()
	logger := s.logger.WithContext(ctx)

	logger.Info("this is a info message")

	s.Equal("00:00:00.000 default info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContext() {
	s.logger.Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger:mocked trace\n00:00:00.000 default info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContextWithChannel() {
	s.logger.WithChannel("channel").Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger:mocked trace\n00:00:00.000 channel info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestInfoWithoutContextWithFields() {
	s.logger.WithFields(mon.Fields{}).Info("this is a info message")
	s.Equal("00:00:00.000 context_missing warn    you should add the context to your logger:mocked trace\n00:00:00.000 default info    this is a info message\n", s.output.String())
}

func (s *ContextEnforcingLoggerTestSuite) TestDebugWithoutContext() {
	s.logger.Debug("this is a debug message")
	s.Empty(s.output.String())
}

func TestContextEnforcingLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(ContextEnforcingLoggerTestSuite))
}
