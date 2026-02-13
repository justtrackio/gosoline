package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type IdentityTestSuite struct {
	suite.Suite
}

func TestIdentityTestSuite(t *testing.T) {
	suite.Run(t, new(IdentityTestSuite))
}

func (s *IdentityTestSuite) TestGetIdentityFromConfig() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "name",
			"env":  "test",
			"tags": map[string]any{
				"project": "prj",
				"family":  "fam",
				"group":   "grp",
			},
		},
	})

	identity, err := cfg.GetAppIdentity(config)
	s.NoError(err)

	expected := cfg.Identity{
		Name: "name",
		Env:  "test",
		Tags: map[string]string{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}

	s.Equal(expected.Name, identity.Name)
	s.Equal(expected.Env, identity.Env)
	s.Equal(expected.Tags, identity.Tags)
}

func (s *IdentityTestSuite) TestPadFromConfig() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "name",
			"env":  "test",
			"tags": map[string]any{
				"project": "prj",
				"family":  "fam",
				"group":   "grp",
			},
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	expected := cfg.Identity{
		Name: "name",
		Env:  "test",
		Tags: map[string]string{
			"project": "prj",
			"family":  "fam",
			"group":   "grp",
		},
	}

	s.Equal(expected.Name, identity.Name)
	s.Equal(expected.Env, identity.Env)
	s.Equal(expected.Tags, identity.Tags)
}

func (s *IdentityTestSuite) TestFormatIdentifier() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
			"tags": map[string]any{
				"project": "gosoline",
			},
			"namespace": "{app.tags.project}.{app.env}.{app.name}",
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	tests := []struct {
		name      string
		pattern   string
		delimiter string
		want      string
		wantErr   bool
	}{
		{
			name:      "simple replacement",
			pattern:   "{app.name}-{app.env}",
			delimiter: "-",
			want:      "my-app-dev",
			wantErr:   false,
		},
		{
			name:      "with tags",
			pattern:   "{app.tags.project}-{app.name}",
			delimiter: "-",
			want:      "gosoline-my-app",
			wantErr:   false,
		},
		{
			name:      "with namespace",
			pattern:   "prefix-{app.namespace}-suffix",
			delimiter: "-",
			want:      "prefix-gosoline-dev-my-app-suffix",
			wantErr:   false,
		},
		{
			name:      "unknown placeholder",
			pattern:   "{app.unknown}-{app.name}",
			delimiter: "-",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "missing tag",
			pattern:   "{app.tags.missing}-{app.name}",
			delimiter: "-",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "no placeholders",
			pattern:   "static-string",
			delimiter: "-",
			want:      "static-string",
			wantErr:   false,
		},
		{
			name:      "empty pattern",
			pattern:   "",
			delimiter: "-",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := identity.Format(tt.pattern, tt.delimiter)
			if tt.wantErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tt.want, got)
			}
		})
	}
}

func (s *IdentityTestSuite) TestToPlaceholders() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
			"tags": map[string]any{
				"project": "gosoline",
			},
			"namespace": "{app.tags.project}.{app.env}.{app.name}",
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	placeholders, err := identity.ToPlaceholders("-")
	s.NoError(err)

	expected := map[string]string{
		"app.name":         "my-app",
		"app.env":          "dev",
		"app.tags.project": "gosoline",
		"app.namespace":    "gosoline-dev-my-app",
	}

	s.Equal(expected, placeholders)
}

func (s *IdentityTestSuite) TestToPlaceholders_EmptyNamespace() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	assert.NoError(s.T(), err)

	placeholders, err := identity.ToPlaceholders("-")
	s.NoError(err)

	s.Equal("my-app", placeholders["app.name"])
	s.Equal("dev", placeholders["app.env"])
	s.Equal("", placeholders["app.namespace"])
}

func (s *IdentityTestSuite) TestFormat_EmptyTagValue() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
			"tags": map[string]any{
				"project": "",
			},
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	_, err = identity.Format("{app.tags.project}-{app.name}", "-")
	s.Error(err)
	s.ErrorContains(err, "resolved to an empty value")
}

func (s *IdentityTestSuite) TestFormat_EmptyNamespaceInPattern() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
		},
	})

	identity := cfg.Identity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	// When namespace is not configured and the pattern uses {app.namespace},
	// Format should return an error because the namespace placeholder resolves to empty.
	_, err = identity.Format("{app.namespace}-{app.name}", "-")
	s.Error(err)
	s.ErrorContains(err, "resolved to an empty value")
}
