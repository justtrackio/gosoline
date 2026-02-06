package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AppIdentityTestSuite struct {
	suite.Suite
}

func TestAppIdentityTestSuite(t *testing.T) {
	suite.Run(t, new(AppIdentityTestSuite))
}

func (s *AppIdentityTestSuite) TestGetAppIdentityFromConfig() {
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

	expected := cfg.AppIdentity{
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

func (s *AppIdentityTestSuite) TestPadFromConfig() {
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

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	s.NoError(err)

	expected := cfg.AppIdentity{
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

func (s *AppIdentityTestSuite) TestFormatIdentifier() {
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

	identity := cfg.AppIdentity{}
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

func (s *AppIdentityTestSuite) TestToPlaceholders() {
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

	identity := cfg.AppIdentity{}
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

func (s *AppIdentityTestSuite) TestToPlaceholders_EmptyNamespace() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "dev",
		},
	})

	identity := cfg.AppIdentity{}
	err := identity.PadFromConfig(config)
	assert.NoError(s.T(), err)

	placeholders, err := identity.ToPlaceholders("-")
	s.NoError(err)

	s.Equal("my-app", placeholders["app.name"])
	s.Equal("dev", placeholders["app.env"])
	s.Equal("", placeholders["app.namespace"])
}
