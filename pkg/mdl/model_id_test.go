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
// FormatLegacyModelIdString() tests (replacement for String())
// =============================================================================

func (s *ModelIdTestSuite) TestFormatLegacyModelIdString_DefaultPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result := mdl.FormatLegacyModelIdString(modelId)
	s.Equal("myProject.myFamily.myGroup.testModel", result)
}

func (s *ModelIdTestSuite) TestFormatLegacyModelIdString_MissingTags_ReturnsEmpty() {
	modelId := mdl.ModelId{
		Name: "testModel",
		// Missing required tags for default pattern
	}

	result := mdl.FormatLegacyModelIdString(modelId)
	// With missing tags, placeholders resolve to empty strings
	s.Equal("...testModel", result)
}

func (s *ModelIdTestSuite) TestFormatLegacyModelIdString_EmptyName() {
	modelId := mdl.ModelId{
		Name: "",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result := mdl.FormatLegacyModelIdString(modelId)
	s.Equal("myProject.myFamily.myGroup.", result)
}

// =============================================================================
// FormatModelIdWithPattern() tests (replacement for Format())
// =============================================================================

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_LegacyPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, mdl.LegacyModelIdPattern)
	s.NoError(err)
	s.Equal("myProject.myFamily.myGroup.testModel", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_WithEnvAndApp() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "production",
		App:  "myApp",
		Tags: map[string]string{
			"project": "myProject",
		},
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.project}-{app.env}-{app.name}-{modelId}")
	s.NoError(err)
	s.Equal("myProject-production-myApp-testModel", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_SinglePlaceholder_ModelId() {
	modelId := mdl.ModelId{
		Name: "testModel",
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "{modelId}")
	s.NoError(err)
	s.Equal("testModel", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_SinglePlaceholder_AppEnv() {
	modelId := mdl.ModelId{
		Env: "staging",
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}")
	s.NoError(err)
	s.Equal("staging", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_SinglePlaceholder_AppName() {
	modelId := mdl.ModelId{
		App: "myApplication",
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.name}")
	s.NoError(err)
	s.Equal("myApplication", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_CustomTags() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"region":     "eu-west-1",
			"team":       "platform",
			"costCenter": "12345",
		},
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.region}-{app.tags.team}-{app.tags.costCenter}-{modelId}")
	s.NoError(err)
	s.Equal("eu-west-1-platform-12345-testModel", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_StaticPattern() {
	modelId := mdl.ModelId{
		Name: "testModel",
	}

	result, err := mdl.FormatModelIdWithPattern(modelId, "my-static-table-name")
	s.NoError(err)
	s.Equal("my-static-table-name", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_DifferentDelimiters() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "dev",
	}

	// Dash delimiter
	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}-{modelId}")
	s.NoError(err)
	s.Equal("dev-testModel", result)

	// Underscore delimiter
	result, err = mdl.FormatModelIdWithPattern(modelId, "{app.env}_{modelId}")
	s.NoError(err)
	s.Equal("dev_testModel", result)

	// Slash delimiter
	result, err = mdl.FormatModelIdWithPattern(modelId, "{app.env}/{modelId}")
	s.NoError(err)
	s.Equal("dev/testModel", result)

	// Colon delimiter
	result, err = mdl.FormatModelIdWithPattern(modelId, "{app.env}:{modelId}")
	s.NoError(err)
	s.Equal("dev:testModel", result)
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_EmptyPattern() {
	modelId := mdl.ModelId{Name: "test"}

	_, err := mdl.FormatModelIdWithPattern(modelId, "")
	s.Error(err)
	s.Contains(err.Error(), "pattern cannot be empty")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_UnknownPlaceholder() {
	modelId := mdl.ModelId{Name: "test"}

	_, err := mdl.FormatModelIdWithPattern(modelId, "{unknown}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_MissingTag() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			// Missing "family" tag
		},
	}

	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.project}-{app.tags.family}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "missing required tags: family")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_MissingMultipleTags() {
	modelId := mdl.ModelId{
		Name: "testModel",
		// Missing all tags
	}

	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.project}-{app.tags.family}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "missing required tags")
	s.Contains(err.Error(), "project")
	s.Contains(err.Error(), "family")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_EmptyEnv() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "", // Empty
	}

	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "app.env but it is empty")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_EmptyApp() {
	modelId := mdl.ModelId{
		Name: "testModel",
		App:  "", // Empty
	}

	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.name}-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "app.name but it is empty")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_StaticTextInPattern() {
	modelId := mdl.ModelId{Name: "test"}

	// Pattern with static text prefix
	_, err := mdl.FormatModelIdWithPattern(modelId, "prefix-{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "static text")
}

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_Error_InconsistentDelimiters() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Pattern with different delimiters
	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}-{modelId}.something")
	s.Error(err)
}

// =============================================================================
// ParseLegacyModelId() tests (replacement for ModelIdFromString())
// =============================================================================

func (s *ModelIdTestSuite) TestParseLegacyModelId_DefaultPattern() {
	modelId, err := mdl.ParseLegacyModelId("myProject.myFamily.myGroup.testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("myProject", modelId.Tags["project"])
	s.Equal("myFamily", modelId.Tags["family"])
	s.Equal("myGroup", modelId.Tags["group"])
}

func (s *ModelIdTestSuite) TestParseLegacyModelId_Roundtrip() {
	original := mdl.ModelId{
		Name: "testModel",
		Tags: map[string]string{
			"project": "myProject",
			"family":  "myFamily",
			"group":   "myGroup",
		},
	}

	str := mdl.FormatLegacyModelIdString(original)
	parsed, err := mdl.ParseLegacyModelId(str)
	s.NoError(err)

	s.Equal(original.Name, parsed.Name)
	s.Equal(original.Tags["project"], parsed.Tags["project"])
	s.Equal(original.Tags["family"], parsed.Tags["family"])
	s.Equal(original.Tags["group"], parsed.Tags["group"])
}

func (s *ModelIdTestSuite) TestParseLegacyModelId_Error_WrongSegmentCount() {
	_, err := mdl.ParseLegacyModelId("only.two.segments")
	s.Error(err)
	s.Contains(err.Error(), "has 3 segments but pattern expects 4")
}

func (s *ModelIdTestSuite) TestParseLegacyModelId_Error_TooManySegments() {
	_, err := mdl.ParseLegacyModelId("one.two.three.four.five")
	s.Error(err)
	s.Contains(err.Error(), "has 5 segments but pattern expects 4")
}

// =============================================================================
// ParseModelIdWithPattern() tests (replacement for ModelIdFromStringWithPattern())
// =============================================================================

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_EnvAndModelId() {
	modelId, err := mdl.ParseModelIdWithPattern("{app.env}-{modelId}", "production-myModel")
	s.NoError(err)

	s.Equal("myModel", modelId.Name)
	s.Equal("production", modelId.Env)
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_AllFields() {
	pattern := "{app.tags.project}-{app.env}-{app.name}-{modelId}"
	modelId, err := mdl.ParseModelIdWithPattern(pattern, "myProject-staging-myApp-testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("staging", modelId.Env)
	s.Equal("myApp", modelId.App)
	s.Equal("myProject", modelId.Tags["project"])
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_CustomTags() {
	// NOTE: When values might contain the delimiter character, use a different delimiter
	// or ensure values don't contain the delimiter. Parsing is inherently ambiguous
	// when values contain the delimiter.
	pattern := "{app.tags.region}_{app.tags.team}_{modelId}"
	modelId, err := mdl.ParseModelIdWithPattern(pattern, "eu-west-1_platform_myModel")
	s.NoError(err)

	s.Equal("myModel", modelId.Name)
	s.Equal("eu-west-1", modelId.Tags["region"])
	s.Equal("platform", modelId.Tags["team"])
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_SinglePlaceholder() {
	modelId, err := mdl.ParseModelIdWithPattern("{modelId}", "justTheModel")
	s.NoError(err)

	s.Equal("justTheModel", modelId.Name)
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_UnderscoreDelimiter() {
	pattern := "{app.env}_{app.name}_{modelId}"
	modelId, err := mdl.ParseModelIdWithPattern(pattern, "dev_myApp_testModel")
	s.NoError(err)

	s.Equal("testModel", modelId.Name)
	s.Equal("dev", modelId.Env)
	s.Equal("myApp", modelId.App)
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_Roundtrip() {
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

	formatted, err := mdl.FormatModelIdWithPattern(original, pattern)
	s.NoError(err)
	s.Equal("myProject-production-myFamily-myGroup-testModel", formatted)

	parsed, err := mdl.ParseModelIdWithPattern(pattern, formatted)
	s.NoError(err)

	s.Equal(original.Name, parsed.Name)
	s.Equal(original.Env, parsed.Env)
	s.Equal(original.Tags["project"], parsed.Tags["project"])
	s.Equal(original.Tags["family"], parsed.Tags["family"])
	s.Equal(original.Tags["group"], parsed.Tags["group"])
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_Error_InvalidPattern() {
	_, err := mdl.ParseModelIdWithPattern("", "test")
	s.Error(err)
	s.Contains(err.Error(), "pattern cannot be empty")
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_Error_WrongSegments() {
	pattern := "{app.env}-{modelId}"
	_, err := mdl.ParseModelIdWithPattern(pattern, "only-one-segment-too-many")
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
			"app.env":              "production",
			"app.name":             "myApp",
			"app.model_id.pattern": "{app.tags.project}.{app.tags.family}.{app.tags.group}.{modelId}",
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
			"app.env":              "production",
			"app.name":             "configApp",
			"app.model_id.pattern": "{app.tags.project}.{app.tags.family}.{modelId}",
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

func (s *ModelIdTestSuite) TestPadFromConfig_MissingPattern_NoError() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":  "production",
			"app.name": "myApp",
			// app.model_id.pattern is missing
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "myProject",
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	// PadFromConfig should succeed even without pattern
	err := modelId.PadFromConfig(config)
	s.NoError(err)

	// Identity fields should be padded
	s.Equal("production", modelId.Env)
	s.Equal("myApp", modelId.App)
	s.Equal("myProject", modelId.Tags["project"])

	// But Format() should fail since pattern is not set
	_, err = modelId.Format()
	s.Error(err)
	s.Contains(err.Error(), "pattern is not set")
}

func (s *ModelIdTestSuite) TestPadFromConfig_InvalidPattern_ReturnsError() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "production",
			"app.name":             "myApp",
			"app.model_id.pattern": "{unknown}-{modelId}", // Unknown placeholder
		},
		stringMaps: map[string]map[string]any{},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	err := modelId.PadFromConfig(config)
	s.Error(err)
	s.Contains(err.Error(), "invalid app.model_id.pattern")
}

func (s *ModelIdTestSuite) TestPadFromConfig_PartialConfig() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "staging",
			"app.model_id.pattern": "{app.env}-{modelId}",
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
// Format() tests
// =============================================================================

func (s *ModelIdTestSuite) TestFormat_AfterPadFromConfig() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "production",
			"app.name":             "myApp",
			"app.model_id.pattern": "{app.tags.project}.{app.tags.family}.{app.tags.group}.{modelId}",
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

	result, err := modelId.Format()
	s.NoError(err)
	s.Equal("myProject.myFamily.myGroup.testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_WithEnvAndAppPattern() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "staging",
			"app.name":             "myApp",
			"app.model_id.pattern": "{app.env}-{app.name}-{modelId}",
		},
		stringMaps: map[string]map[string]any{},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	err := modelId.PadFromConfig(config)
	s.NoError(err)

	result, err := modelId.Format()
	s.NoError(err)
	s.Equal("staging-myApp-testModel", result)
}

func (s *ModelIdTestSuite) TestFormat_WithoutPadFromConfig_ReturnsError() {
	modelId := mdl.ModelId{
		Name: "testModel",
		Env:  "production",
		Tags: map[string]string{
			"project": "myProject",
		},
	}

	// Format without calling PadFromConfig should fail
	_, err := modelId.Format()
	s.Error(err)
	s.Contains(err.Error(), "pattern is not set")
	s.Contains(err.Error(), "PadFromConfig")
}

func (s *ModelIdTestSuite) TestFormat_MissingRequiredTag_ReturnsError() {
	config := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "production",
			"app.model_id.pattern": "{app.tags.project}.{app.tags.family}.{modelId}",
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "myProject",
				// family is missing
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	err := modelId.PadFromConfig(config)
	s.NoError(err)

	_, err = modelId.Format()
	s.Error(err)
	s.Contains(err.Error(), "missing required tags: family")
}

func (s *ModelIdTestSuite) TestFormat_PatternNotOverwrittenBySecondPadFromConfig() {
	config1 := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "production",
			"app.model_id.pattern": "{app.env}-{modelId}",
		},
		stringMaps: map[string]map[string]any{},
	}

	config2 := &mockConfigProvider{
		strings: map[string]string{
			"app.env":              "staging",
			"app.model_id.pattern": "{app.tags.project}.{modelId}", // Different pattern
		},
		stringMaps: map[string]map[string]any{
			"app.tags": {
				"project": "myProject",
			},
		},
	}

	modelId := mdl.ModelId{
		Name: "testModel",
	}

	// First pad
	err := modelId.PadFromConfig(config1)
	s.NoError(err)

	// Second pad should not overwrite the pattern
	err = modelId.PadFromConfig(config2)
	s.NoError(err)

	// Should use the first pattern
	result, err := modelId.Format()
	s.NoError(err)
	s.Equal("production-testModel", result)
}

// =============================================================================
// Pattern validation edge cases
// =============================================================================

func (s *ModelIdTestSuite) TestPatternValidation_EmptyTagKey() {
	modelId := mdl.ModelId{Name: "test"}

	// app.tags. without a key after it
	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.}")
	s.Error(err)
	s.Contains(err.Error(), "unknown placeholder")
}

func (s *ModelIdTestSuite) TestPatternValidation_MultiCharDelimiter() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Two-character delimiter is not allowed
	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}--{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "delimiter must be a single character")
}

func (s *ModelIdTestSuite) TestPatternValidation_AlphanumericDelimiter() {
	modelId := mdl.ModelId{Name: "test", Env: "dev"}

	// Letter as delimiter is not allowed
	_, err := mdl.FormatModelIdWithPattern(modelId, "{app.env}x{modelId}")
	s.Error(err)
	s.Contains(err.Error(), "delimiter must be non-alphanumeric")
}

// =============================================================================
// Edge cases with special characters in values
// =============================================================================

func (s *ModelIdTestSuite) TestFormatModelIdWithPattern_ValuesWithSpecialChars() {
	modelId := mdl.ModelId{
		Name: "test-model-v2",
		Env:  "us-east-1",
		Tags: map[string]string{
			"project": "my_project",
		},
	}

	// Values can contain the delimiter character
	result, err := mdl.FormatModelIdWithPattern(modelId, "{app.tags.project}.{app.env}.{modelId}")
	s.NoError(err)
	s.Equal("my_project.us-east-1.test-model-v2", result)
}

func (s *ModelIdTestSuite) TestParseModelIdWithPattern_ValuesWithDelimiterChar() {
	// When parsing, values containing the delimiter cause issues
	// This is expected behavior - the delimiter splits the string
	_, err := mdl.ParseModelIdWithPattern(
		"{app.env}-{modelId}",
		"us-east-1-my-model", // "us-east-1" and "my-model" both contain "-"
	)
	s.Error(err)
	// "us-east-1-my-model" splits by "-" into 5 segments: ["us", "east", "1", "my", "model"]
	s.Contains(err.Error(), "has 5 segments but pattern expects 2")
}
