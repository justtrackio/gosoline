package test

import (
	"testing"
)

type StreamTestCaseInput struct {
	Attributes map[string]interface{}
	Body       interface{}
}

type StreamTestCaseOutput struct {
	Model              interface{}
	ExpectedAttributes map[string]interface{}
	ExpectedBody       interface{}
}

type StreamTestCase struct {
	Input  map[string][]StreamTestCaseInput
	Output map[string][]StreamTestCaseOutput
}

type TestingSuiteStream interface {
	TestingSuite
	SetupTestCases() []StreamTestCase
	TestStreamCase(app AppUnderTest, testCase StreamTestCase)
}

func RunStreamTestSuite(t *testing.T, suite TestingSuiteStream) {
	suite.SetT(t)
	testCases := suite.SetupTestCases()

	for _, tc := range testCases {
		RunTestCase(t, suite, func(appUnderTest AppUnderTest) {
			suite.TestStreamCase(appUnderTest, tc)
		})
	}
}

type StreamTestSuite struct {
	Suite
}

func (s StreamTestSuite) TestStreamCase(app AppUnderTest, testCase StreamTestCase) {
	for inputName, data := range testCase.Input {
		input := s.Env().StreamInput(inputName)

		for _, d := range data {
			input.Publish(d.Body, d.Attributes)
		}

		input.Stop()
	}

	app.WaitDone()

	for outputName, data := range testCase.Output {
		output := s.Env().StreamOutput(outputName)

		for i, d := range data {
			model := d.Model
			attrs := output.Unmarshal(i, model)

			s.Equal(d.ExpectedAttributes, attrs, "attributes do not match")
			s.Equal(d.ExpectedBody, model, "body does not match")
		}
	}
}
