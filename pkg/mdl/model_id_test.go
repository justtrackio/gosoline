package mdl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestModelIdTestSuite(t *testing.T) {
	suite.Run(t, new(ModelIdTestSuite))
}

type ModelIdTestSuite struct {
	suite.Suite
}

// =============================================================================
// String() tests
// =============================================================================

func (s *ModelIdTestSuite) TestString_DefaultPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result := modelId.String()
	s.Equal("myProject.myFamily.myGroup.testModel", result)
}

func (s *ModelIdTestSuite) TestString_MissingTags_ReturnsInvalid() {
	modelId := mdl.ModelId{
		Name: "testModel",
		// Missing required tags for default pattern
	}

	result := modelId.String()
	s.Equal("<invalid:testModel>", result)
}

func (s *ModelIdTestSuite) TestString_EmptyName() {
	modelId := mdl.ModelId{
		Name: "",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result := modelId.String()
	s.Equal("myProject.myFamily.myGroup.", result)
}

// =============================================================================
// Format() tests
// =============================================================================

func (s *ModelIdTestSuite) TestFormat_DefaultPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result, err := modelId.Format(mdl.DefaultModelIdPattern)
	s.NoError(err)
	s.Equal("myProject.myFamily.myGroup.testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_WithEnvAndApp() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "production",
		App:  "myApp",
		Tags: map[string]string{
			"project": "myProject",
		},
	}

	result, err := modelId.Format("{app.tags.project}-{app.env}-{app.name}-{modelId}")
	s.NoError(err)
	s.Equal("myProject-production-myApp-testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_SinglePlaceholder_ModelId() {
	modelId := mdl.ModelId{
		Name: "testModel",
	}

	result, err := modelId.Format("{modelId}")
	s.NoError(err)
	s.Equal("testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_SinglePlaceholder_AppEnv() {
	modelId := mdl.ModelId{
		Env: "staging",
	}

	result, err := modelId.Format("{app.env}")
	s.NoError(err)
	s.Equal("staging", result)
}

func (s *ModelIdTestSuite) TestFormat_SinglePlaceholder_AppName() {
	modelId := mdl.ModelId{
		App: "myApplication",
	}

	result, err := modelId.Format("{app.name}")
	s.NoError(err)
	s.Equal("myApplication", result)
}

func (s *ModelIdTestSuite) TestFormat_CustomTags() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"region":     "eu-west-1",
			"team":       "platform",
			"costCenter": "12345",
		},
	}

	result, err := modelId.Format("{app.tags.region}-{app.tags.team}-{app.tags.costCenter}-{modelId}")
	s.NoError(err)
	s.Equal("eu-west-1-platform-12345-testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_StaticPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
	}

	result, err := modelId.Format("my-static-table-name")
	s.NoError(err)
	s.Equal("my-static-table-name", result)
}

func (s *ModelIdTestSuite) TestFormat_DifferentDelimiters() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "dev",
	}

	// Dash delimiter
	result, err := modelId.Format("{app.env}-{modelId}")
	s.NoError(err)
	s.Equal("dev-testModel", result)

	// Underscore delimiter
	result, err = modelId.Format("{app.env}_{modelId}")
	s.NoError(err)
	s.Equal("dev_testModel", result)

	// Slash delimiter
	result, err = modelId.Format("{app.env}/{modelId}")
	s.NoError(err)
	s.Equal("dev/testModel", result)

	// Colon delimiter
	result, err = modelId.Format("{app.env}:{modelId}")
	s.NoError(err)
	s.Equal("dev:testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_Error_EmptyPattern() {
	modelId := mdl.ModelId{Name: "test"}

	_, err := modelId.Format("")
	s.Error(err)
	s.Contains(err.Error(), "pattern cannot be empty")
}

func (s *ModelIdTestSuite) TestFormat_Error_UnknownPlaceholder() {
	modelId := mdl.ModelId{Name: "test"}

	_, err := modelId.Format("{unknown}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder")
}

func (s *ModelIdTestSuite) TestFormat_Error_MissingTag() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			// Missing "family" tag
		},
	}

	_, err := modelId.Format("{app.tags.project}-{app.tags.family}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "missing required tags: family")
}

func (s *ModelIdTestSuite) TestFormat_Error_MissingMultipleTags() {
	modelId := mdl.ModelId{
		Name: "testModel",
		// Missing all tags
	}

	_, err := modelId.Format("{app.tags.project}-{app.tags.family}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "missing required tags")
	s.Contains(err.Error(), "project")
	s.Contains(err.Error(), "family")
}

func (s *ModelIdTestSuite) TestFormat_Error_EmptyEnv() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "", // Empty
	}

	_, err := modelId.Format("{app.env}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "app.env but it is empty")
}

func (s *ModelIdTestSuite) TestFormat_Error_EmptyApp() {
	modelId := mdl.ModelId{
		Name: "testModel",
		App:  "", // Empty
	}

	_, err := modelId.Format("{app.name}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "app.name but it is empty")
}

func (s *ModelIdTestSuite) TestFormat_Error_StaticTextInPattern() {
	modelId := mdl.ModelId{Name: "test"}

	// Pattern with static text prefix
	_, err := modelId.Format("prefix-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "static text")
}

func (s *ModelIdTestSuite) TestFormat_Error_InconsistentDelimiters() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Pattern with different delimiters
	_, err := modelId.Format("{app.env}-{modelId}.something")
	s.Error(err)
}

// =============================================================================
// ModelIdFromString() tests
// =============================================================================

func (s *ModelIdTestSuite) TestModelIdFromString_DefaultPattern() {
	modelId, err := mdl.ModelIdFromString("myProject.myFamily.myGroup.testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("myProject", modelId.Tags["project"])
	s.Equal("myFamily", modelId.Tags["family"])
	s.Equal("myGroup", modelId.Tags["group"])
}

func (s *ModelIdTestSuite) TestModelIdFromString_Roundtrip() {
	original := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	str := original.String()
	parsed, err := mdl.ModelIdFromString(str)
	s.NoError(err)

	s.Equal(original.Name, parsed.Name)
	s.Equal(original.Tags["project"], parsed.Tags["project"])
	s.Equal(original.Tags["family"], parsed.Tags["family"])
	s.Equal(original.Tags["group"], parsed.Tags["group"])
}

func (s *ModelIdTestSuite) TestModelIdFromString_Error_WrongSegmentCount() {
	_, err := mdl.ModelIdFromString("only.two.segments")
	s.Error(err)
	s.Contains(err.Error(), "has 3 segments but pattern expects 4")
}

func (s *ModelIdTestSuite) TestModelIdFromString_Error_TooManySegments() {
	_, err := mdl.ModelIdFromString("one.two.three.four.five")
	s.Error(err)
	s.Contains(err.Error(), "has 5 segments but pattern expects 4")
}

// =============================================================================
// ModelIdFromStringWithPattern() tests
// =============================================================================

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_EnvAndModelId() {
	modelId, err := mdl.ModelIdFromStringWithPattern("{app.env}-{modelId}", "production-myModel")
	s.NoError(err)

	s.Equal("myModel", modelId.Name)
	s.Equal("production", modelId.Env)
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_AllFields() {
	pattern := "{app.tags.project}-{app.env}-{app.name}-{modelId}"
	modelId, err := mdl.ModelIdFromStringWithPattern(pattern, "myProject-staging-myApp-testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("staging", modelId.Env)
	s.Equal("myApp", modelId.App)
	s.Equal("myProject", modelId.Tags["project"])
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_CustomTags() {
	// NOTE: When values might contain the delimiter character, use a different delimiter
	// or ensure values don't contain the delimiter. Parsing is inherently ambiguous
	// when values contain the delimiter.
	pattern := "{app.tags.region}_{app.tags.team}_{modelId}"
	modelId, err := mdl.ModelIdFromStringWithPattern(pattern, "eu-west-1_platform_myModel")
	s.NoError(err)

	s.Equal("myModel", modelId.Name)
	s.Equal("eu-west-1", modelId.Tags["region"])
	s.Equal("platform", modelId.Tags["team"])
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_SinglePlaceholder() {
	modelId, err := mdl.ModelIdFromStringWithPattern("{modelId}", "justTheModel")
	s.NoError(err)

	s.Equal("justTheModel", modelId.Name)
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_UnderscoreDelimiter() {
	pattern := "{app.env}_{app.name}_{modelId}"
	modelId, err := mdl.ModelIdFromStringWithPattern(pattern, "dev_myApp_testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("dev", modelId.Env)
	s.Equal("myApp", modelId.App)
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_Roundtrip() {
	pattern := "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{modelId}"
	original := mdl.ModelId{
		Name: "testModel",
		Env:  "production",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	formatted, err := original.Format(pattern)
	s.NoError(err)
	s.Equal("myProject-production-myFamily-myGroup-testModel", formatted)

	parsed, err := mdl.ModelIdFromStringWithPattern(pattern, formatted)
	s.NoError(err)

	s.Equal(original.Name, parsed.Name)
	s.Equal(original.Env, parsed.Env)
	s.Equal(original.Tags["project"], parsed.Tags["project"])
	s.Equal(original.Tags["family"], parsed.Tags["family"])
	s.Equal(original.Tags["group"], parsed.Tags["group"])
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_Error_InvalidPattern() {
	_, err := mdl.ModelIdFromStringWithPattern("", "test")
	s.Error(err)
	s.Contains(err.Error(), "invalid pattern")
}

func (s *ModelIdTestSuite) TestModelIdFromStringWithPattern_Error_WrongSegments() {
	pattern := "{app.env}-{modelId}"
	_, err := mdl.ModelIdFromStringWithPattern(pattern, "only-one-segment-too-many")
	s.Error(err)
	// "only-one-segment-too-many" splits by "-" into 5 segments
	s.Contains(err.Error(), "has 5 segments but pattern expects 2")
}

// =============================================================================
// PadFromConfig() tests
// =============================================================================

type mockConfigProvider struct {
	strings    map[string]string
	stringMaps map[string]map[string]any
}

func (m *mockConfigProvider) GetString(key string, optionalDefault ...string) (string, error) {
	if val, ok := m.strings[key]; ok {
		return val, nil
	}
	if len(optionalDefault) > 0 {
		return optionalDefault[0], nil
	}

	return "", assert.AnError
}

func (m *mockConfigProvider) GetStringMap(key string, optionalDefault ...map[string]any) (map[string]any, error) {
	if val, ok := m.stringMaps[key]; ok {
		return val, nil
	}
	if len(optionalDefault) > 0 {
		return optionalDefault[0], nil
	}

	return nil, assert.AnError
}

func (s *ModelIdTestSuite) TestPadFromConfig_AllFields() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":  "production",
			"app.name": "myApp",
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "myProject",
				"family":  "myFamily",
				"group":   "myGroup",
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	err := modelId.PadFromConfig(config)
	s.NoError(err)

	s.Equal("production", modelId.Env)
	s.Equal("myApp", modelId.App)
	s.Equal("myProject", modelId.Tags["project"])
	s.Equal("myFamily", modelId.Tags["family"])
	s.Equal("myGroup", modelId.Tags["group"])
}

func (s *ModelIdTestSuite) TestPadFromConfig_ExistingValuesNotOverwritten() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":  "production",
			"app.name": "configApp",
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "configProject",
				"family":  "configFamily",
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "existing-env",
		App:  "existing-app",
		Tags: map[string]string{
			"project": "existing-project",
		},
	}

	err := modelId.PadFromConfig(config)
	s.NoError(err)

	// Existing values should not be overwritten
	s.Equal("existing-env", modelId.Env)
	s.Equal("existing-app", modelId.App)
	s.Equal("existing-project", modelId.Tags["project"])
	// But missing tags should be filled
	s.Equal("configFamily", modelId.Tags["family"])
}

func (s *ModelIdTestSuite) TestPadFromConfig_MissingConfigKeys() {
	config := &mockConfigProvider{
		strings:    map[string]string{},
		stringMaps: map[string]map[string]any{},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	// Should not error even if config keys are missing
	err := modelId.PadFromConfig(config)
	s.NoError(err)

	// Values should remain empty
	s.Equal("", modelId.Env)
	s.Equal("", modelId.App)
	s.NotNil(modelId.Tags) // Tags should be initialized
}

func (s *ModelIdTestSuite) TestPadFromConfig_PartialConfig() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env": "staging",
			// app.name is missing
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "myProject",
				// family and group are missing
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	err := modelId.PadFromConfig(config)
	s.NoError(err)

	s.Equal("staging", modelId.Env)
	s.Equal("", modelId.App) // Not in config
	s.Equal("myProject", modelId.Tags["project"])
}

// =============================================================================
// Pattern validation edge cases
// =============================================================================

func (s *ModelIdTestSuite) TestPatternValidation_EmptyTagKey() {
	modelId := mdl.ModelId{Name: "test"}

	// app.tags. without a key after it
	_, err := modelId.Format("{app.tags.}")
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder")
}

func (s *ModelIdTestSuite) TestPatternValidation_MultiCharDelimiter() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Two-character delimiter is not allowed
	_, err := modelId.Format("{app.env}--{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "delimiter must be a single character")
}

func (s *ModelIdTestSuite) TestPatternValidation_AlphanumericDelimiter() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Letter as delimiter is not allowed
	_, err := modelId.Format("{app.env}x{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "delimiter must be non-alphanumeric")
}

// =============================================================================
// Edge cases with special characters in values
// =============================================================================

func (s *ModelIdTestSuite) TestFormat_ValuesWithSpecialChars() {
	modelId := mdl.ModelId{
		Name: "test-model-v2",
		Env:  "us-east-1",
		Tags: map[string]string{
			"project": "my_project",
		},
	}

	// Values can contain the delimiter character
	result, err := modelId.Format("{app.tags.project}.{app.env}.{modelId}")
	s.NoError(err)
	s.Equal("my_project.us-east-1.test-model-v2", result)
}

func (s *ModelIdTestSuite) TestParse_ValuesWithDelimiterChar() {
	// When parsing, values containing the delimiter cause issues
	// This is expected behavior - the delimiter splits the string
	_, err := mdl.ModelIdFromStringWithPattern(
		"{app.env}-{modelId}",
		"us-east-1-my-model", // "us-east-1" and "my-model" both contain "-"
	)
	s.Error(err)
	// "us-east-1-my-model" splits by "-" into 5 segments: ["us", "east", "1", "my", "model"]
	s.Contains(err.Error(), "has 5 segments but pattern expects 2")
}
