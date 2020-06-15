package test_test

import (
	"github.com/applike/gosoline/pkg/test"
	"testing"
)

type Case1 struct {
	test.Suite
	foo string
}

func (c *Case1) SetupSuite() []test.SuiteOption {
	return nil
}

func (c *Case1) TestSomething(_ test.AppUnderTest) {
	c.True(true, "assert")
}

func TestRunCase(t *testing.T) {
	test.RunCase(t, new(Case1))
}
