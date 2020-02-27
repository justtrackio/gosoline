package mon_test

import (
	"bytes"
	"context"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jonboulle/clockwork"
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
	s.base = mon.NewLoggerWithInterfaces(s.clock, s.output)

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

func (s *ContextEnforcingLoggerTestSuite) TestDebugWithoutContext() {
	s.logger.Debug("this is a info message")
	s.Empty(s.output.String())
}

func TestContextEnforcingLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(ContextEnforcingLoggerTestSuite))
}
