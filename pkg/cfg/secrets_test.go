package cfg_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/require"
)

func TestSecretSchemaRootedEmptyContainersAndDecodeFailure(t *testing.T) {
	type row struct {
		Value  string `cfg:"value,secret"`
		Public string `cfg:"public"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"rows":   []any{},
		"groups": map[string]any{},
	})
	var rows []row
	require.NoError(t, config.UnmarshalKey("rows", &rows))
	var groups map[string]*row
	require.NoError(t, config.UnmarshalKey("groups", &groups))
	require.True(t, config.IsSecret("rows[11].value"))
	require.False(t, config.IsSecret("rows.11.value"))
	require.False(t, config.IsSecret("rows[11].public"))
	require.True(t, config.IsSecret(`groups.tenant\.literal.value`))
	require.False(t, config.IsSecret("groups[0].value"))
	require.False(t, config.IsSecret("other.value"))
	var invalid struct {
		Opaque int `cfg:"opaque,secret"`
	}
	require.NoError(t, config.Option(cfg.WithConfigSetting("invalid.opaque", "synthetic-not-an-int")))
	require.Error(t, config.UnmarshalKey("invalid", &invalid))
	output, _ := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-not-an-int")
}

func TestSecretSchemaSelectorBoundariesAndPrefixOptions(t *testing.T) {
	type Embedded struct {
		Embedded string `cfg:"embedded,secret"`
	}
	type child struct {
		Opaque string `cfg:"Opaque,SeCrEt,NoDecode,NoCast"`
		Public string `cfg:"public"`
	}
	type settings struct {
		Embedded
		Child   *child `cfg:"ignored,prefix=SDK_"`
		Literal string `cfg:"literal\\.opaque,secret"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"provider": map[string]any{"SDK_Opaque": "synthetic-prefixed", "SDK_public": "visible-prefix", "embedded": "synthetic-embedded", "literal\\.opaque": "synthetic-literal", "literal": map[string]any{"opaque": "visible-nested"}},
	})
	var decoded settings
	require.NoError(t, config.UnmarshalKey("provider", &decoded))
	require.True(t, config.IsSecret("provider.SDK_Opaque"))
	require.False(t, config.IsSecret("provider.sdk_opaque"))
	require.False(t, config.IsSecret("provider.SDK_public"))
	require.True(t, config.IsSecret(`provider.literal\.opaque`))
	require.False(t, config.IsSecret("provider.literal.opaque"))
	require.True(t, config.IsSecret("provider.embedded"))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, messages, "cfg provider.SDK_public=visible-prefix")
	require.Contains(t, messages, "cfg provider.literal.opaque=visible-nested")
}

func TestSecretSchemaRecursiveTypesAndConcurrentRegistration(t *testing.T) {
	type node struct {
		Value    string           `cfg:"value,secret"`
		Children map[string]*node `cfg:"children"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	var output map[string]*node
	require.Error(t, config.UnmarshalKey("tree", &output))
	require.True(t, config.IsSecret("tree.first.children.second.children.third.value"))
	var wg sync.WaitGroup
	errors := make(chan error, 20)
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var nodes map[string]*node
			errors <- config.UnmarshalKey(fmt.Sprintf("empty%d", i), &nodes)
			_ = config.IsSecret("tree.first.children.second.value")
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.Error(t, err)
	}
	require.True(t, config.IsSecret("empty19.future.value"))
}

func TestSecretSchemaFullDefaultsMappingAndInstanceIsolation(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"shared": map[string]any{"opaque": "synthetic-shared", "access_key_id": "public-shared"}})
	var decoded debugCredentials
	require.NoError(t, config.UnmarshalDefaults(&decoded, cfg.UnmarshalWithDefaultsFromKey("shared", ".")))
	require.Equal(t, "synthetic-shared", decoded.Opaque)
	require.True(t, config.IsSecret("shared.opaque"))
	require.False(t, config.IsSecret("opaque"))
	other := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.False(t, other.IsSecret("shared.opaque"))
	output, _ := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-shared")
	require.Contains(t, output, "public-shared")
}

func TestDebugConfigNestedTypedMapsSlicesAndLiteralDotBoundary(t *testing.T) {
	type rows []*debugCredentials
	type groups map[string]rows
	native := groups{"tenant.with.dot": {{Opaque: "synthetic-nested", AccessKeyID: "public-nested"}}}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigSetting("nested", native)))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-nested")
	require.Contains(t, messages, "cfg nested.tenant.with.dot.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg nested.tenant.with.dot.0.access_key_id=public-nested")
	require.True(t, config.IsSecret(`nested.tenant\.with\.dot[0].opaque`))
	require.Equal(t, "synthetic-nested", native["tenant.with.dot"][0].Opaque)

	type collision struct {
		Opaque string `cfg:"literal.opaque,secret"`
	}
	require.NoError(t, config.Option(cfg.WithConfigSetting("collision", &collision{Opaque: "synthetic-collision"})))
	require.NoError(t, config.Option(cfg.WithConfigSetting(`collision.literal\.opaque`, "public-literal")))
	output, messages = captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-collision")
	require.Contains(t, messages, "cfg collision.literal.opaque=[REDACTED]")
	require.Contains(t, output, "public-literal")
	require.False(t, config.IsSecret(`collision.literal\.opaque`))
}

func TestRootUnmarshalRegistersTagsWithoutDuplicatingSettings(t *testing.T) {
	for _, root := range []string{"", "."} {
		t.Run(root, func(t *testing.T) {
			type settings struct {
				Opaque string `cfg:"api_auth_keys,secret"`
			}
			env := cfg.NewMemoryEnvProvider(map[string]string{"APP_API_AUTH_KEYS": "synthetic-env-root"})
			config := cfg.NewWithInterfaces(env, map[string]any{"api_auth_keys": "synthetic-stored-root", "public": "visible-root"})
			require.NoError(t, config.Option(cfg.WithEnvKeyPrefix("app"), cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer)))
			var decoded settings
			require.NoError(t, config.UnmarshalKey(root, &decoded))
			require.Equal(t, "synthetic-env-root", decoded.Opaque)
			require.True(t, config.IsSecret("api_auth_keys"))
			require.NotContains(t, config.AllSettings(), "")
			require.Equal(t, "visible-root", config.AllSettings()["public"])
			value, err := config.GetString("public")
			require.NoError(t, err)
			require.Equal(t, "visible-root", value)
			output, messages := captureDebugConfig(t, config)
			require.NotContains(t, output, "synthetic-")
			require.Contains(t, messages, "cfg api_auth_keys=[REDACTED]")
		})
	}
}

func TestIndexedTagRetainsOtherElementsSecretAnnotations(t *testing.T) {
	type row struct {
		Token string `cfg:"token,secret"`
	}
	type publicRow struct {
		Token string `cfg:"token"`
	}
	type settings struct {
		Rows  []row     `cfg:"rows"`
		First publicRow `cfg:"rows[0]"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigSetting("provider", &settings{
		Rows:  []row{{Token: "synthetic-first"}, {Token: "synthetic-second"}},
		First: publicRow{Token: "visible-first"},
	})))
	require.True(t, config.IsSecret("provider.rows[1].token"))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-second")
	require.Contains(t, messages, "cfg provider.rows.1.token=[REDACTED]")
}

func TestDebugConfigProtectsBracketContainingDynamicMapKeys(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	clients := map[string]debugCredentials{
		"tenant[0]": {Opaque: "synthetic-bracket", AccessKeyID: "public-bracket"},
	}
	groups := map[string][]debugCredentials{
		"tenant[0]": {{Opaque: "synthetic-bracket-slice", AccessKeyID: "public-bracket-slice"}},
	}
	deepGroups := map[string][][]debugCredentials{
		"tenant[0]": {{{Opaque: "synthetic-bracket-deep", AccessKeyID: "public-bracket-deep"}}},
	}
	matrix := [][]debugCredentials{{{Opaque: "synthetic-matrix", AccessKeyID: "public-matrix"}}}
	deeperGroups := map[string][][][]debugCredentials{
		"tenant[0][1]": {{{{Opaque: "synthetic-bracket-deeper", AccessKeyID: "public-bracket-deeper"}}}},
	}
	cube := [][][]debugCredentials{{{{Opaque: "synthetic-cube", AccessKeyID: "public-cube"}}}}
	require.NoError(t, config.Option(
		cfg.WithConfigSetting("clients", clients),
		cfg.WithConfigSetting("groups", groups),
		cfg.WithConfigSetting("deep_groups", deepGroups),
		cfg.WithConfigSetting("matrix", matrix),
		cfg.WithConfigSetting("deeper_groups", deeperGroups),
		cfg.WithConfigSetting("cube", cube),
	))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-bracket")
	require.NotContains(t, output, "synthetic-matrix")
	require.NotContains(t, output, "synthetic-cube")
	require.Contains(t, messages, "cfg clients.tenant[0].opaque=[REDACTED]")
	require.Contains(t, messages, "cfg clients.tenant[0].access_key_id=public-bracket")
	require.Contains(t, messages, "cfg groups.tenant[0].0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg groups.tenant[0].0.access_key_id=public-bracket-slice")
	require.Contains(t, messages, "cfg deep_groups.tenant[0].0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg deep_groups.tenant[0].0.0.access_key_id=public-bracket-deep")
	require.Contains(t, messages, "cfg matrix.0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg matrix.0.0.access_key_id=public-matrix")
	require.True(t, config.IsSecret("matrix[0][0].opaque"))
	require.False(t, config.IsSecret("matrix[0][0].access_key_id"))
	require.Contains(t, messages, "cfg deeper_groups.tenant[0][1].0.0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg deeper_groups.tenant[0][1].0.0.0.access_key_id=public-bracket-deeper")
	require.Contains(t, messages, "cfg cube.0.0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg cube.0.0.0.access_key_id=public-cube")
	require.True(t, config.IsSecret("cube[0][0][0].opaque"))
	require.False(t, config.IsSecret("cube[0][0][0].access_key_id"))
	require.Equal(t, "synthetic-bracket", clients["tenant[0]"].Opaque)
	require.Equal(t, "synthetic-bracket-slice", groups["tenant[0]"][0].Opaque)
	require.Equal(t, "synthetic-bracket-deep", deepGroups["tenant[0]"][0][0].Opaque)
	require.Equal(t, "synthetic-matrix", matrix[0][0].Opaque)
	require.Equal(t, "synthetic-bracket-deeper", deeperGroups["tenant[0][1]"][0][0][0].Opaque)
	require.Equal(t, "synthetic-cube", cube[0][0][0].Opaque)
}

func TestDebugConfigProtectsBareBracketDynamicMapKeys(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	clients := map[string]debugCredentials{
		"[0]":    {Opaque: "synthetic-bare-bracket", AccessKeyID: "public-bare-bracket"},
		"[0][1]": {Opaque: "synthetic-bare-bracket-many", AccessKeyID: "public-bare-bracket-many"},
	}
	groups := map[string][]debugCredentials{
		"[0]": {{Opaque: "synthetic-bare-bracket-slice", AccessKeyID: "public-bare-bracket-slice"}},
	}
	require.NoError(t, config.Option(cfg.WithConfigSetting("clients", clients), cfg.WithConfigSetting("groups", groups)))
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-bare-bracket")
	require.Contains(t, messages, "cfg clients.[0].opaque=[REDACTED]")
	require.Contains(t, messages, "cfg clients.[0].access_key_id=public-bare-bracket")
	require.Contains(t, messages, "cfg clients.[0][1].opaque=[REDACTED]")
	require.Contains(t, messages, "cfg clients.[0][1].access_key_id=public-bare-bracket-many")
	require.Contains(t, messages, "cfg groups.[0].0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg groups.[0].0.access_key_id=public-bare-bracket-slice")
	require.Equal(t, "synthetic-bare-bracket", clients["[0]"].Opaque)
	require.Equal(t, "synthetic-bare-bracket-slice", groups["[0]"][0].Opaque)
}
