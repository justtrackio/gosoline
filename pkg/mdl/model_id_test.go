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
		{
			name:     "static prefix with placeholder",
			tags:     map[string]any{},
			pattern:  "prefix-{app.env}",
			expected: "prefix-test.testEvent",
		},
		{
			name:     "placeholders with dash delimiter",
			tags:     map[string]any{"project": "myProject"},
			pattern:  "{app.tags.project}-{app.env}",
			expected: "myProject-test.testEvent",
		},
		{
			name:     "placeholder with underscore delimiter",
			tags:     map[string]any{"project": "myProject"},
			pattern:  "{app.tags.project}_{app.name}",
			expected: "myProject_myApp.testEvent",
		},
		{
			name:     "static prefix and suffix with placeholder",
			tags:     map[string]any{},
			pattern:  "pre-{app.env}-suf",
			expected: "pre-test-suf.testEvent",
		},
		{
			name:     "mixed static and multiple placeholders",
			tags:     map[string]any{"project": "myProject"},
			pattern:  "ns-{app.tags.project}.{app.env}-live",
			expected: "ns-myProject.test-live.testEvent",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().GetString("app.env").Return("test", nil)
			config.EXPECT().GetString("app.name").Return("myApp", nil)
			config.EXPECT().GetStringMap("app.tags").Return(tc.tags, nil)
			config.EXPECT().Get("app.model_id.domain_pattern").Return(tc.pattern, nil)

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

func (s *ModelIdTestSuite) TestPadFromConfig_MissingTagInPattern() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetString("app.name").Return("myApp", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)
	config.EXPECT().Get("app.model_id.domain_pattern").Return("{app.tags.project}.{app.env}", nil)

	modelId := &mdl.ModelId{Name: "testEvent"}
	err := modelId.PadFromConfig(config)
	s.Error(err)
	s.ErrorContains(err, "project")
	s.ErrorContains(err, "required by domain pattern")
}

func (s *ModelIdTestSuite) TestPadFromConfig_TagsPresentInPattern() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetString("app.name").Return("myApp", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{"project": "myProject"}, nil)
	config.EXPECT().Get("app.model_id.domain_pattern").Return("{app.tags.project}.{app.env}", nil)

	modelId := &mdl.ModelId{Name: "testEvent"}
	err := modelId.PadFromConfig(config)
	s.NoError(err)
	s.Equal("myProject.test.testEvent", modelId.String())
}

func (s *ModelIdTestSuite) TestPadFromConfig_NoTagPlaceholders() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetString("app.name").Return("myApp", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)
	config.EXPECT().Get("app.model_id.domain_pattern").Return("{app.env}", nil)

	modelId := &mdl.ModelId{Name: "testEvent"}
	err := modelId.PadFromConfig(config)
	s.NoError(err)
	s.Equal("test.testEvent", modelId.String())
}

func (s *ModelIdTestSuite) TestPadFromConfig_MultipleMissingTags() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetString("app.name").Return("myApp", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)
	config.EXPECT().Get("app.model_id.domain_pattern").Return("{app.tags.project}.{app.tags.family}", nil)

	modelId := &mdl.ModelId{Name: "testEvent"}
	err := modelId.PadFromConfig(config)
	s.Error(err)
	s.ErrorContains(err, "required by domain pattern")
}

func (s *ModelIdTestSuite) TestPadFromConfig_PartialTagsMissing() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.env").Return("test", nil)
	config.EXPECT().GetString("app.name").Return("myApp", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{"project": "myProject"}, nil)
	config.EXPECT().Get("app.model_id.domain_pattern").Return("{app.tags.project}.{app.tags.family}", nil)

	modelId := &mdl.ModelId{Name: "testEvent"}
	err := modelId.PadFromConfig(config)
	s.Error(err)
	s.ErrorContains(err, "family")
	s.ErrorContains(err, "required by domain pattern")
}
