package suite_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

type BaseTestSuite struct {
	suite.Suite
	called bool
}

func TestBaseTestSuite(t *testing.T) {
	var s BaseTestSuite
	suite.Run(t, &s)
	assert.True(t, s.called)
}

func (s *BaseTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithSharedEnvironment(),
	}
}

func (s *BaseTestSuite) TestBase() {
	s.called = true
}
