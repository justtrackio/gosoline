package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
)

type ResourceIdentifierTestSuite struct {
	suite.Suite
}

func TestResourceIdentifierTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceIdentifierTestSuite))
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_AllEmpty() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "prod",
			"tags": map[string]any{
				"project": "prj",
				"family":  "fam",
			},
		},
	})

	ri := cfg.ResourceIdentifier{}
	err := ri.PadFromConfig(config)
	s.NoError(err)

	s.Equal("my-app", ri.Application)
	s.Equal("prod", ri.Env)
	s.Equal(cfg.Tags{"project": "prj", "family": "fam"}, ri.Tags)
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_PresetApplicationNotOverwritten() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "global-app",
			"env":  "prod",
		},
	})

	ri := cfg.ResourceIdentifier{Application: "other-service"}
	err := ri.PadFromConfig(config)
	s.NoError(err)

	// Pre-set Application must not be overwritten by app.name
	s.Equal("other-service", ri.Application)
	s.Equal("prod", ri.Env)
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_PresetEnvNotOverwritten() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "prod",
		},
	})

	ri := cfg.ResourceIdentifier{Env: "staging"}
	err := ri.PadFromConfig(config)
	s.NoError(err)

	s.Equal("staging", ri.Env)
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_TagsMerged() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
			"env":  "prod",
			"tags": map[string]any{
				"project": "from-config",
				"family":  "from-config",
			},
		},
	})

	// Per-resource tag wins over config tag for "project"; config fills "family"
	ri := cfg.ResourceIdentifier{Tags: cfg.Tags{"project": "override"}}
	err := ri.PadFromConfig(config)
	s.NoError(err)

	s.Equal("override", ri.Tags["project"], "per-resource tag must win")
	s.Equal("from-config", ri.Tags["family"], "missing key must be filled from config")
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_MissingAppName() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env": "prod",
		},
	})

	ri := cfg.ResourceIdentifier{}
	err := ri.PadFromConfig(config)
	s.Error(err)
}

func (s *ResourceIdentifierTestSuite) TestPadFromConfig_MissingAppEnv() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "my-app",
		},
	})

	ri := cfg.ResourceIdentifier{}
	err := ri.PadFromConfig(config)
	s.Error(err)
}

func (s *ResourceIdentifierTestSuite) TestToIdentity_MapsApplicationToName() {
	ri := cfg.ResourceIdentifier{
		Application: "user-service",
		Env:         "prod",
		Tags:        cfg.Tags{"project": "prj"},
	}

	identity := ri.ToIdentity()

	s.Equal("user-service", identity.Name, "Application must map to Identity.Name")
	s.Equal("prod", identity.Env)
	s.Equal(cfg.Tags{"project": "prj"}, identity.Tags)
}

func (s *ResourceIdentifierTestSuite) TestToIdentity_NoNamespace() {
	ri := cfg.ResourceIdentifier{
		Application: "svc",
		Env:         "dev",
	}

	identity := ri.ToIdentity()

	// Namespace is a formatting concern populated by Identity.PadFromConfig;
	// ToIdentity does not set it.
	s.Empty(identity.Namespace)
}

func (s *ResourceIdentifierTestSuite) TestToIdentity_ThenPadFromConfig_PopulatesNamespace() {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name":      "my-app",
			"env":       "dev",
			"namespace": "{app.tags.project}.{app.env}.{app.name}",
			"tags": map[string]any{
				"project": "gosoline",
			},
		},
	})

	ri := cfg.ResourceIdentifier{Application: "other-app"}
	err := ri.PadFromConfig(config)
	s.NoError(err)

	identity := ri.ToIdentity()
	err = identity.PadFromConfig(config)
	s.NoError(err)

	name, err := identity.Format("{app.namespace}-{queueId}", "-", map[string]string{"queueId": "events"})
	s.NoError(err)
	// Namespace uses other-app (from ResourceIdentifier.Application), not global my-app
	s.Equal("gosoline-dev-other-app-events", name)
}

func (s *ResourceIdentifierTestSuite) TestUnmarshalFromConfig_Embedded() {
	// Verify that embedding ResourceIdentifier into a config struct flattens keys.
	type inputConfig struct {
		cfg.ResourceIdentifier
		QueueId string `cfg:"queue_id"`
	}

	config := cfg.New(map[string]any{
		"stream": map[string]any{
			"input": map[string]any{
				"my-input": map[string]any{
					"application": "user-service",
					"env":         "prod",
					"tags": map[string]any{
						"project": "prj",
					},
					"queue_id": "user-events",
				},
			},
		},
	})

	ic := inputConfig{}
	err := config.UnmarshalKey("stream.input.my-input", &ic)
	s.NoError(err)

	s.Equal("user-service", ic.Application)
	s.Equal("prod", ic.Env)
	s.Equal(cfg.Tags{"project": "prj"}, ic.Tags)
	s.Equal("user-events", ic.QueueId)
}
