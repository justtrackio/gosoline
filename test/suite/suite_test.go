//go:build integration
// +build integration

package suite_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

type SuiteTestSuite struct {
	suite.Suite
	testCount int
}

func (s *SuiteTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithConfigFile("config.dist.yml"),
		suite.WithSharedEnvironment(),
		suite.WithTestCaseWhitelist("TestExpectedToRun"),
		suite.WithTestCaseRepeatCount(2),
	}
}

func (s *SuiteTestSuite) TestExpectedToRun() {
	s.testCount++
}

func (s *SuiteTestSuite) TestExpectedToSkip() {
	s.FailNow("should've been skipped")
}

func TestSuiteTestSuite(t *testing.T) {
	s := &SuiteTestSuite{}
	suite.Run(t, s)
	assert.Equal(t, 2, s.testCount)
}
