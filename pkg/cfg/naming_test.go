package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
)

func TestNamingTemplate_Validate(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		resourcePhs []string
		wantErr     string
	}{
		{
			name:    "valid pattern with identity placeholders",
			pattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}",
			wantErr: "",
		},
		{
			name:        "valid pattern with resource placeholder",
			pattern:     "{app.tags.project}-{app.env}-{queueId}",
			resourcePhs: []string{"queueId"},
			wantErr:     "",
		},
		{
			name:    "valid pattern with arbitrary tag",
			pattern: "{app.tags.region}-{app.tags.costCenter}-{app.env}",
			wantErr: "",
		},
		{
			name:    "valid pattern with custom tag",
			pattern: "{app.tags.team}-{app.env}",
			wantErr: "",
		},
		{
			name:    "unknown placeholder - old style project",
			pattern: "{project}-{app.env}",
			wantErr: `unknown placeholder(s) {project} in pattern "{project}-{app.env}"`,
		},
		{
			name:    "unknown placeholder - typo app.tag instead of app.tags",
			pattern: "{app.tag.project}-{app.env}",
			wantErr: `unknown placeholder(s) {app.tag.project} in pattern "{app.tag.project}-{app.env}"`,
		},
		{
			name:    "unknown placeholder - old style env",
			pattern: "{app.tags.project}-{env}",
			wantErr: `unknown placeholder(s) {env} in pattern "{app.tags.project}-{env}"`,
		},
		{
			name:    "multiple unknown placeholders",
			pattern: "{family}-{group}-{app.env}",
			wantErr: `unknown placeholder(s) {family}, {group} in pattern "{family}-{group}-{app.env}"`,
		},
		{
			name:        "unknown resource placeholder not registered",
			pattern:     "{app.env}-{topicId}",
			resourcePhs: []string{"queueId"}, // registered queueId but used topicId
			wantErr:     `unknown placeholder(s) {topicId} in pattern "{app.env}-{topicId}"`,
		},
		{
			name:    "unclosed placeholder",
			pattern: "{app.env}-{app.name",
			wantErr: `unclosed placeholder in pattern "{app.env}-{app.name"`,
		},
		{
			name:    "empty placeholder",
			pattern: "{app.env}-{}",
			wantErr: `empty placeholder {} in pattern "{app.env}-{}"`,
		},
		{
			name:    "empty tag key",
			pattern: "{app.tags.}-{app.env}",
			wantErr: `unknown placeholder(s) {app.tags.} in pattern "{app.tags.}-{app.env}"`,
		},
		{
			name:    "no placeholders is valid",
			pattern: "static-name",
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := cfg.NewNamingTemplate(tt.pattern, tt.resourcePhs...)
			err := tmpl.Validate()

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestNamingTemplate_RequiredTags(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected []string
	}{
		{
			name:     "all common tags",
			pattern:  "{app.tags.project}-{app.tags.family}-{app.tags.group}",
			expected: []string{"project", "family", "group"},
		},
		{
			name:     "only project",
			pattern:  "{app.tags.project}-{app.env}",
			expected: []string{"project"},
		},
		{
			name:     "no tags",
			pattern:  "{app.env}-{app.name}",
			expected: nil,
		},
		{
			name:     "duplicate tags counted once",
			pattern:  "{app.tags.project}-{app.tags.project}",
			expected: []string{"project"},
		},
		{
			name:     "arbitrary custom tags",
			pattern:  "{app.tags.region}-{app.tags.costCenter}-{app.tags.team}",
			expected: []string{"region", "costCenter", "team"},
		},
		{
			name:     "mixed common and custom tags",
			pattern:  "{app.tags.project}-{app.tags.region}-{app.env}",
			expected: []string{"project", "region"},
		},
		{
			name:     "empty tag key not included",
			pattern:  "{app.tags.}-{app.env}",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := cfg.NewNamingTemplate(tt.pattern)
			got := tmpl.RequiredTags()
			assert.ElementsMatch(t, tt.expected, got)
		})
	}
}

func TestNamingTemplate_RequiresAppName(t *testing.T) {
	assert.True(t, cfg.NewNamingTemplate("{app.name}-test").RequiresAppName())
	assert.False(t, cfg.NewNamingTemplate("{app.env}-test").RequiresAppName())
}

func TestNamingTemplate_RequiresEnv(t *testing.T) {
	assert.True(t, cfg.NewNamingTemplate("{app.env}-test").RequiresEnv())
	assert.False(t, cfg.NewNamingTemplate("{app.name}-test").RequiresEnv())
}

func TestNamingTemplate_Expand(t *testing.T) {
	identity := cfg.AppIdentity{
		Name: "myapp",
		Env:  "prod",
		Tags: cfg.AppTags{
			"project": "myproject",
			"family":  "myfamily",
			"group":   "mygroup",
		},
	}

	tmpl := cfg.NewNamingTemplate("{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}", "queueId")
	tmpl.WithResourceValue("queueId", "orders")

	result := tmpl.Expand(identity)
	assert.Equal(t, "myproject-prod-myfamily-mygroup-orders", result)
}

func TestNamingTemplate_Expand_DynamicTags(t *testing.T) {
	identity := cfg.AppIdentity{
		Name: "myapp",
		Env:  "prod",
		Tags: cfg.AppTags{
			"region":     "eu-west-1",
			"costCenter": "CC123",
			"team":       "platform",
		},
	}

	tmpl := cfg.NewNamingTemplate("{app.tags.region}-{app.tags.costCenter}-{app.tags.team}-{app.env}")

	result := tmpl.Expand(identity)
	assert.Equal(t, "eu-west-1-CC123-platform-prod", result)
}

func TestNamingTemplate_Expand_MissingTagReturnsEmpty(t *testing.T) {
	// Expand does NOT validate - it just expands (returns empty for missing tags)
	identity := cfg.AppIdentity{
		Env: "prod",
		// no tags
	}

	tmpl := cfg.NewNamingTemplate("{app.tags.project}-{app.env}")

	result := tmpl.Expand(identity)
	assert.Equal(t, "-prod", result) // missing tag becomes empty
}

func TestNamingTemplate_ValidateAndExpand(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		resourcePhs  []string
		resourceVals map[string]string
		identity     cfg.AppIdentity
		want         string
		wantErr      string
	}{
		{
			name:         "success with all fields",
			pattern:      "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{queueId}",
			resourcePhs:  []string{"queueId"},
			resourceVals: map[string]string{"queueId": "orders"},
			identity: cfg.AppIdentity{
				Name: "myapp",
				Env:  "prod",
				Tags: cfg.AppTags{
					"project": "myproject",
					"family":  "myfamily",
					"group":   "mygroup",
				},
			},
			want:    "myproject-prod-myfamily-mygroup-orders",
			wantErr: "",
		},
		{
			name:         "success with minimal pattern",
			pattern:      "{app.env}-{queueId}",
			resourcePhs:  []string{"queueId"},
			resourceVals: map[string]string{"queueId": "orders"},
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			want:    "prod-orders",
			wantErr: "",
		},
		{
			name:    "success with custom tags",
			pattern: "{app.tags.region}-{app.tags.costCenter}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
				Tags: cfg.AppTags{
					"region":     "eu-west-1",
					"costCenter": "CC123",
				},
			},
			want:    "eu-west-1-CC123-prod",
			wantErr: "",
		},
		{
			name:    "missing required tag",
			pattern: "{app.tags.project}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
				// no tags
			},
			wantErr: "missing required tags: project",
		},
		{
			name:    "missing custom required tag",
			pattern: "{app.tags.region}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
				Tags: cfg.AppTags{
					"project": "myproject", // has project but not region
				},
			},
			wantErr: "missing required tags: region",
		},
		{
			name:    "missing multiple required tags",
			pattern: "{app.tags.project}-{app.tags.family}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			wantErr: "missing required tags: family, project",
		},
		{
			name:    "missing app.name",
			pattern: "{app.name}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			wantErr: "naming pattern requires app.name but it is empty",
		},
		{
			name:    "missing app.env",
			pattern: "{app.env}-test",
			identity: cfg.AppIdentity{
				Name: "myapp",
			},
			wantErr: "naming pattern requires app.env but it is empty",
		},
		{
			name:    "unknown placeholder error - old style project",
			pattern: "{project}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			wantErr: `unknown placeholder(s) {project} in pattern "{project}-{app.env}"`,
		},
		{
			name:    "unknown placeholder error - old style env",
			pattern: "{env}-{app.tags.project}",
			identity: cfg.AppIdentity{
				Env:  "prod",
				Tags: cfg.AppTags{"project": "myproject"},
			},
			wantErr: `unknown placeholder(s) {env} in pattern "{env}-{app.tags.project}"`,
		},
		{
			name:    "unknown placeholder error - typo",
			pattern: "{app.tag.project}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			wantErr: `unknown placeholder(s) {app.tag.project} in pattern "{app.tag.project}-{app.env}"`,
		},
		{
			name:    "empty tag key error",
			pattern: "{app.tags.}-{app.env}",
			identity: cfg.AppIdentity{
				Env: "prod",
			},
			wantErr: `unknown placeholder(s) {app.tags.} in pattern "{app.tags.}-{app.env}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := cfg.NewNamingTemplate(tt.pattern, tt.resourcePhs...)
			for k, v := range tt.resourceVals {
				tmpl.WithResourceValue(k, v)
			}

			got, err := tmpl.ValidateAndExpand(tt.identity)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
