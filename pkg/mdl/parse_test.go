package mdl_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

func TestParseModelIdTestSuite(t *testing.T) {
	suite.Run(t, new(ParseModelIdTestSuite))
}

type ParseModelIdTestSuite struct {
	suite.Suite
}

func (s *ParseModelIdTestSuite) TestParseModelId_Success() {
	type testCase struct {
		name          string
		domainPattern string
		input         string
		expectedEnv   string
		expectedApp   string
		expectedTags  map[string]string
		expectedName  string
	}

	cases := []testCase{
		{
			name:          "single placeholder app.env",
			domainPattern: "{app.env}",
			input:         "production.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{},
			expectedName:  "users",
		},
		{
			name:          "single placeholder app.name",
			domainPattern: "{app.name}",
			input:         "myapp.users",
			expectedEnv:   "",
			expectedApp:   "myapp",
			expectedTags:  map[string]string{},
			expectedName:  "users",
		},
		{
			name:          "single tag placeholder",
			domainPattern: "{app.tags.project}",
			input:         "myproject.users",
			expectedEnv:   "",
			expectedApp:   "",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
		{
			name:          "two placeholders",
			domainPattern: "{app.tags.project}.{app.env}",
			input:         "myproject.production.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
		{
			name:          "three placeholders",
			domainPattern: "{app.tags.project}.{app.env}.{app.name}",
			input:         "myproject.production.myapp.users",
			expectedEnv:   "production",
			expectedApp:   "myapp",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
		{
			name:          "multiple tags",
			domainPattern: "{app.tags.project}.{app.tags.family}.{app.env}",
			input:         "myproject.myfamily.production.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags: map[string]string{
				"project": "myproject",
				"family":  "myfamily",
			},
			expectedName: "users",
		},
		{
			name:          "complex pattern with all types",
			domainPattern: "{app.tags.project}.{app.env}.{app.name}.{app.tags.group}",
			input:         "myproject.production.myapp.mygroup.users",
			expectedEnv:   "production",
			expectedApp:   "myapp",
			expectedTags: map[string]string{
				"project": "myproject",
				"group":   "mygroup",
			},
			expectedName: "users",
		},
		{
			name:          "static pattern",
			domainPattern: "staticdomain",
			input:         "users",
			expectedEnv:   "",
			expectedApp:   "",
			expectedTags:  map[string]string{},
			expectedName:  "users",
		},
		{
			name:          "static prefix with placeholder",
			domainPattern: "prefix-{app.env}",
			input:         "prefix-production.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{},
			expectedName:  "users",
		},
		{
			name:          "placeholders with dash delimiter",
			domainPattern: "{app.tags.project}-{app.env}",
			input:         "myproject-production.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
		{
			name:          "placeholder with underscore delimiter",
			domainPattern: "{app.tags.project}_{app.name}",
			input:         "myproject_myapp.users",
			expectedEnv:   "",
			expectedApp:   "myapp",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
		{
			name:          "static prefix and suffix with placeholder",
			domainPattern: "pre-{app.env}-suf",
			input:         "pre-production-suf.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{},
			expectedName:  "users",
		},
		{
			name:          "mixed static and multiple placeholders",
			domainPattern: "ns-{app.tags.project}.{app.env}-live",
			input:         "ns-myproject.production-live.users",
			expectedEnv:   "production",
			expectedApp:   "",
			expectedTags:  map[string]string{"project": "myproject"},
			expectedName:  "users",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(tc.domainPattern, nil)

			modelId, err := mdl.ParseModelId(config, tc.input)

			s.NoError(err)
			s.Equal(tc.expectedEnv, modelId.Env, "Env mismatch")
			s.Equal(tc.expectedApp, modelId.App, "App mismatch")
			s.Equal(tc.expectedName, modelId.Name, "Name mismatch")
			s.Equal(tc.expectedTags, modelId.Tags, "Tags mismatch")
			s.Equal(tc.domainPattern, modelId.DomainPattern, "DomainPattern should be set")
		})
	}
}

func (s *ParseModelIdTestSuite) TestParseModelId_ErrorMissingConfig() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(nil, fmt.Errorf("config key not found"))

	_, err := mdl.ParseModelId(config, "production.users")

	s.Error(err)
	s.Contains(err.Error(), "app.model_id.domain_pattern must be set")
}

func (s *ParseModelIdTestSuite) TestParseModelId_ErrorEmptyPattern() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return("", nil)

	_, err := mdl.ParseModelId(config, "production.users")

	s.Error(err)
	s.Contains(err.Error(), "must not be empty")
}

func (s *ParseModelIdTestSuite) TestParseModelId_ErrorInvalidPattern() {
	type testCase struct {
		name          string
		domainPattern string
		expectedError string
	}

	cases := []testCase{
		{
			name:          "unknown placeholder",
			domainPattern: "{app.unknown}",
			expectedError: "unknown placeholder",
		},
		{
			name:          "empty tag key",
			domainPattern: "{app.tags.}",
			expectedError: "invalid app.model_id.domain_pattern: tag key is empty for placeholder {app.tags.}",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(tc.domainPattern, nil)

			_, err := mdl.ParseModelId(config, "production.users")

			s.Error(err)
			s.Contains(err.Error(), tc.expectedError)
		})
	}
}

func (s *ParseModelIdTestSuite) TestParseModelId_ErrorInputMismatch() {
	type testCase struct {
		name          string
		domainPattern string
		input         string
		expectedError string
	}

	cases := []testCase{
		{
			name:          "too few segments",
			domainPattern: "{app.tags.project}.{app.env}",
			input:         "production.users",
			expectedError: "does not match domain pattern",
		},
		{
			name:          "empty string",
			domainPattern: "{app.env}",
			input:         "",
			expectedError: "does not match domain pattern",
		},
		{
			name:          "wrong static prefix",
			domainPattern: "prefix-{app.env}",
			input:         "wrong-production.users",
			expectedError: "does not match domain pattern",
		},
		{
			name:          "missing model name separator",
			domainPattern: "{app.env}",
			input:         "production",
			expectedError: "does not match domain pattern",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(tc.domainPattern, nil)

			_, err := mdl.ParseModelId(config, tc.input)

			s.Error(err)
			s.Contains(err.Error(), tc.expectedError)
		})
	}
}

func (s *ParseModelIdTestSuite) TestParseModelId_RoundTrip() {
	// Test that we can parse a ModelId string and get back the same values
	type testCase struct {
		name          string
		domainPattern string
		originalId    mdl.ModelId
	}

	cases := []testCase{
		{
			name:          "simple env pattern",
			domainPattern: "{app.env}",
			originalId: mdl.ModelId{
				Name: "users",
				Env:  "production",
				Tags: map[string]string{},
			},
		},
		{
			name:          "complex pattern",
			domainPattern: "{app.tags.project}.{app.env}.{app.name}",
			originalId: mdl.ModelId{
				Name: "users",
				Env:  "production",
				App:  "myapp",
				Tags: map[string]string{"project": "myproject"},
			},
		},
		{
			name:          "multiple tags",
			domainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
			originalId: mdl.ModelId{
				Name: "users",
				Tags: map[string]string{
					"project": "myproject",
					"family":  "myfamily",
					"group":   "mygroup",
				},
			},
		},
		{
			name:          "static prefix with placeholder",
			domainPattern: "prefix-{app.env}",
			originalId: mdl.ModelId{
				Name: "users",
				Env:  "production",
				Tags: map[string]string{},
			},
		},
		{
			name:          "placeholders with dash delimiter",
			domainPattern: "{app.tags.project}-{app.env}",
			originalId: mdl.ModelId{
				Name: "users",
				Env:  "production",
				Tags: map[string]string{"project": "myproject"},
			},
		},
		{
			name:          "mixed static and placeholders",
			domainPattern: "ns-{app.tags.project}.{app.env}-live",
			originalId: mdl.ModelId{
				Name: "users",
				Env:  "production",
				Tags: map[string]string{"project": "myproject"},
			},
		},
		{
			name:          "placeholder with underscore delimiter",
			domainPattern: "{app.tags.project}_{app.name}",
			originalId: mdl.ModelId{
				Name: "users",
				App:  "myapp",
				Tags: map[string]string{"project": "myproject"},
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			// Setup original ModelId with domain pattern
			tc.originalId.DomainPattern = tc.domainPattern

			// Generate the string representation
			originalString := tc.originalId.String()

			// Parse it back
			config := mocks.NewConfig(s.T())
			config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(tc.domainPattern, nil)

			parsedId, err := mdl.ParseModelId(config, originalString)

			s.NoError(err)
			s.Equal(tc.originalId.Name, parsedId.Name, "Name should match")
			s.Equal(tc.originalId.Env, parsedId.Env, "Env should match")
			s.Equal(tc.originalId.App, parsedId.App, "App should match")
			s.Equal(tc.originalId.Tags, parsedId.Tags, "Tags should match")
			s.Equal(tc.domainPattern, parsedId.DomainPattern, "DomainPattern should match")

			// Verify the parsed ModelId generates the same string
			parsedString := parsedId.String()
			s.Equal(originalString, parsedString, "Round-trip string should match")
		})
	}
}

func (s *ParseModelIdTestSuite) TestParseModelId_EdgeCases() {
	type testCase struct {
		name          string
		domainPattern string
		input         string
		shouldError   bool
		expectedError string
	}

	cases := []testCase{
		{
			name:          "single segment with no placeholders",
			domainPattern: "static",
			input:         "users",
			shouldError:   false,
		},
		{
			name:          "empty segments",
			domainPattern: "{app.env}",
			input:         ".users",
			shouldError:   true,
			expectedError: "does not match domain pattern",
		},
		{
			name:          "static pattern with dots in input matches model name with dots",
			domainPattern: "static",
			input:         "static.users",
			shouldError:   false,
		},
		{
			name:          "model name with dots in placeholder pattern",
			domainPattern: "{app.env}",
			input:         "production.my.complex.model",
			shouldError:   false,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			config := mocks.NewConfig(s.T())
			config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return(tc.domainPattern, nil)

			modelId, err := mdl.ParseModelId(config, tc.input)

			if tc.shouldError {
				s.Error(err)
				if tc.expectedError != "" {
					s.Contains(err.Error(), tc.expectedError)
				}
			} else {
				s.NoError(err)
				// Verify the parsed ModelId is valid (has expected number of fields)
				s.NotNil(modelId)
			}
		})
	}
}

func (s *ParseModelIdTestSuite) TestParseModelId_StaticPattern() {
	// Test that static patterns (no placeholders) work correctly
	// Note: Static patterns expect the input to be a single segment (just the model name)
	// because they have 0 placeholders, so expected parts = 0 + 1 = 1
	config := mocks.NewConfig(s.T())
	config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return("staticdomain", nil)

	modelId, err := mdl.ParseModelId(config, "users")

	s.NoError(err)
	s.Equal("users", modelId.Name)
	s.Equal("", modelId.Env)
	s.Equal("", modelId.App)
	s.Empty(modelId.Tags)
	s.Equal("staticdomain", modelId.DomainPattern)

	// Verify it can generate the same string back
	result := modelId.String()
	s.Equal("staticdomain.users", result)
}

func (s *ParseModelIdTestSuite) TestParseModelId_PreservesUnusedFields() {
	// Test that parsing only sets fields referenced in the pattern,
	// leaving other fields at their zero values
	config := mocks.NewConfig(s.T())
	config.EXPECT().Get(mdl.ConfigKeyModelIdDomainPattern).Return("{app.env}", nil)

	modelId, err := mdl.ParseModelId(config, "production.users")

	s.NoError(err)
	s.Equal("production", modelId.Env)
	s.Equal("", modelId.App, "App should be empty since pattern doesn't use it")
	s.Empty(modelId.Tags, "Tags should be empty since pattern doesn't use them")
}
