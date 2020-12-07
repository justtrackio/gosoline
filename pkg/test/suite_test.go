package test_test

import (
	"github.com/applike/gosoline/pkg/test"
	"testing"
)

type Case1 struct {
	test.Suite
}

func (c *Case1) SetupSuite() []test.SuiteOption {
	return nil
}

func (c *Case1) TestSomething(_ test.AppUnderTest) {
	c.True(true, "assert")
}

func TestRunCase(t *testing.T) {
	test.RunSuite(t, new(Case1))
}
