package matcher_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ContextMatcherTestSuite struct {
	suite.Suite
	mockSvc *MockService
}

type MockService struct {
	mock.Mock
}

func (m *MockService) Process(ctx any) string {
	args := m.Called(ctx)

	return args.String(0)
}

func TestRunContextMatcherTestSuite(t *testing.T) {
	suite.Run(t, new(ContextMatcherTestSuite))
}

func (s *ContextMatcherTestSuite) SetupTest() {
	s.mockSvc = new(MockService)
	s.mockSvc.On("Process", matcher.Context).Return("success")
}

func (s *ContextMatcherTestSuite) Test_MatchesContext() {
	result := s.mockSvc.Process(context.Background())
	s.Equal("success", result)

	result = s.mockSvc.Process(context.TODO())
	s.Equal("success", result)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result = s.mockSvc.Process(ctx)
	s.Equal("success", result)
}

func (s *ContextMatcherTestSuite) Test_DoesNotMatchNonContext() {
	s.Panics(func() {
		s.mockSvc.Process("test")
	})

	s.Panics(func() {
		s.mockSvc.Process(nil)
	})

	s.Panics(func() {
		s.mockSvc.Process(123)
	})
}
