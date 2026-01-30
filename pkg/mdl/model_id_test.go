package mdl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

func TestModelIdTestSuite(t *testing.T) {
	suite.Run(t, new(ModelIdTestSuite))
}

type ModelIdTestSuite struct {
	suite.Suite

	config *mocks.Config
}

func (s *ModelIdTestSuite) SetupTest() {
	s.config = mocks.NewConfig(s.T())
}

func (s *ModelIdTestSuite) TestString() {
	type testCase struct {
		name     string
		tags     map[string]any
		pattern  string
		expected string
	}

	cases := []testCase{
		{
			name:     "no tags",
			tags:     map[string]any{},
			pattern:  "{app.name}",
			expected: "myApp.testEvent",
		},
		{
			name:     "with env",
			tags:     map[string]any{},
			pattern:  "{app.env}",
			expected: "test.testEvent",
		},
		{
			name:     "with tags",
			tags:     map[string]any{"project": "myProject"},
			pattern:  "{app.tags.project}",
			expected: "myProject.testEvent",
		},
		{
			name:     "with mixed",
			tags:     map[string]any{"project": "myProject"},
			pattern:  "{app.tags.project}.{app.env}",
			expected: "myProject.test.testEvent",
		},
		{
			name:     "static pattern",
			tags:     map[string]any{},
			pattern:  "static.domain",
			expected: "static.domain.testEvent",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().GetString("app.env").Return("test", nil)
			config.EXPECT().GetString("app.name").Return("myApp", nil)
			config.EXPECT().GetStringMap("app.tags").Return(tc.tags, nil)
			config.EXPECT().GetString("app.model_id.domain_pattern").Return(tc.pattern, nil)

			modelId := &mdl.ModelId{
				Name: "testEvent",
			}
			err := modelId.PadFromConfig(config)
			s.NoError(err)

			str := modelId.String()
			s.Equal(tc.expected, str)
		})
	}
}
