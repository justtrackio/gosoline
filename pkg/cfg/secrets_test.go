package cfg_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	_ "github.com/justtrackio/gosoline/pkg/currency"
	_ "github.com/justtrackio/gosoline/pkg/db"
	_ "github.com/justtrackio/gosoline/pkg/httpserver/auth"
	_ "github.com/justtrackio/gosoline/pkg/kafka/connection"
	_ "github.com/justtrackio/gosoline/pkg/otel"
	_ "github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/require"
)

func TestDebugConfigProtectsFailedDecode(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	var invalid struct {
		Opaque int `cfg:"opaque,secret"`
	}
	require.NoError(t, config.Option(cfg.WithConfigSetting("invalid.opaque", "synthetic-not-an-int")))
	require.Error(t, config.UnmarshalKey("invalid", &invalid))
	output, _ := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-not-an-int")
}

func TestDebugConfigPrefixOptions(t *testing.T) {
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
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, messages, "cfg provider.SDK_public=visible-prefix")
	require.Contains(t, messages, "cfg provider.literal.opaque=visible-nested")
}

func TestDebugConfigFullDefaultsSourceMapping(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"shared": map[string]any{"opaque": "synthetic-shared", "access_key_id": "public-shared"}})
	var decoded debugCredentials
	require.NoError(t, config.UnmarshalDefaults(&decoded, cfg.UnmarshalWithDefaultsFromKey("shared", ".")))
	require.Equal(t, "synthetic-shared", decoded.Opaque)
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
	require.Contains(t, messages, "cfg deeper_groups.tenant[0][1].0.0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg deeper_groups.tenant[0][1].0.0.0.access_key_id=public-bracket-deeper")
	require.Contains(t, messages, "cfg cube.0.0.0.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg cube.0.0.0.access_key_id=public-cube")
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

func TestDebugConfigProtectsConcreteTaggedValuesInUntypedMaps(t *testing.T) {
	type wrapper struct {
		Value any `cfg:"value"`
	}
	for _, test := range []struct {
		name  string
		value any
		path  string
	}{
		{"struct", debugCredentials{Opaque: "synthetic-concrete-secret", AccessKeyID: "public-concrete"}, "client.opaque"},
		{"pointer", &debugCredentials{Opaque: "synthetic-concrete-secret", AccessKeyID: "public-concrete"}, "client.opaque"},
		{"list", []any{debugCredentials{Opaque: "synthetic-concrete-secret", AccessKeyID: "public-concrete"}}, "client.0.opaque"},
		{"typed_wrapper", wrapper{Value: debugCredentials{Opaque: "synthetic-concrete-secret", AccessKeyID: "public-concrete"}}, "client.value.opaque"},
		{"typed_wrapper_map", wrapper{Value: map[string]any{"tenant.key[0]": debugCredentials{Opaque: "synthetic-concrete-secret", AccessKeyID: "public-concrete"}}}, "client.value.tenant.key[0].opaque"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
			require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{"client": test.value})))
			output, messages := captureDebugConfig(t, config)
			require.NotContains(t, output, "synthetic-concrete-secret")
			require.Contains(t, messages, "cfg "+test.path+"=[REDACTED]")
			require.Contains(t, output, "public-concrete")
			stored, err := config.Get("client")
			require.NoError(t, err)
			require.Equal(t, test.value, stored)
		})
	}
}

func TestConcreteSecretAnnotationsStayScopedToMapKeysAndIndexes(t *testing.T) {
	type publicCredentials struct {
		Opaque string `cfg:"opaque"`
	}
	type nativeMap map[string]any
	native := nativeMap{
		"tenant.with.dot[0]": debugCredentials{Opaque: "synthetic-native-private", AccessKeyID: "public-native"},
		"tenant.with.dot[1]": publicCredentials{Opaque: "visible-native-sibling"},
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"clients": map[string]any{
			"private":  debugCredentials{Opaque: "synthetic-key-private", AccessKeyID: "public-key"},
			"public":   publicCredentials{Opaque: "visible-key-sibling"},
			"changing": publicCredentials{Opaque: "visible-before-change"},
		},
		"rows": []any{
			debugCredentials{Opaque: "synthetic-index-private", AccessKeyID: "public-index"},
			publicCredentials{Opaque: "visible-index-sibling"},
		},
		"empty": map[string]debugCredentials{},
	}), cfg.WithConfigSetting("native", native)))
	require.NoError(t, config.Option(cfg.WithConfigMap(map[string]any{
		"clients": map[string]any{"changing": debugCredentials{Opaque: "synthetic-updated-private", AccessKeyID: "public-updated"}},
	})))
	native["future"] = publicCredentials{Opaque: "visible-future"}
	output, messages := captureDebugConfig(t, config)
	for _, secret := range []string{"synthetic-key-private", "synthetic-index-private", "synthetic-native-private", "synthetic-updated-private"} {
		require.NotContains(t, output, secret)
	}
	require.Contains(t, messages, "cfg clients.public.opaque=visible-key-sibling")
	require.Contains(t, messages, "cfg rows.1.opaque=visible-index-sibling")
	require.Contains(t, messages, "cfg native.tenant.with.dot[1].opaque=visible-native-sibling")
	require.Contains(t, messages, "cfg native.future.opaque=visible-future")
	require.Equal(t, "synthetic-native-private", native["tenant.with.dot[0]"].(debugCredentials).Opaque)
}

func TestDebugConfigProtectsConcreteTaggedValuesInInitialMaps(t *testing.T) {
	settings := map[string]any{
		"client":            debugCredentials{Opaque: "synthetic-initial-secret", AccessKeyID: "public-initial"},
		"literal.client[0]": debugCredentials{Opaque: "synthetic-initial-literal-secret", AccessKeyID: "public-initial-literal"},
	}
	for _, test := range []struct {
		name   string
		config cfg.Config
	}{
		{"New", cfg.New(settings)},
		{"NewWithInterfaces", cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), settings)},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, messages := captureDebugConfig(t, test.config)
			require.NotContains(t, output, "synthetic-initial-secret")
			require.Contains(t, messages, "cfg client.opaque=[REDACTED]")
			require.Contains(t, messages, "cfg client.access_key_id=public-initial")
			require.NotContains(t, output, "synthetic-initial-literal-secret")
			require.Contains(t, messages, "cfg literal.client[0].opaque=[REDACTED]")
			require.Contains(t, messages, "cfg literal.client[0].access_key_id=public-initial-literal")
			require.Equal(t, settings["literal.client[0]"], test.config.AllSettings()["literal.client[0]"])
			stored, err := test.config.Get("client")
			require.NoError(t, err)
			require.Equal(t, settings["client"], stored)
		})
	}
}

func TestDebugConfigProtectsConcreteDefaultSourceSubschema(t *testing.T) {
	type output struct {
		Client any `cfg:"client"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"shared_value": "synthetic-concrete-default-private",
	})
	destination := output{Client: debugCredentials{}}
	require.NoError(t, config.UnmarshalDefaults(&destination,
		cfg.UnmarshalWithDefaultsFromKey("shared_value", "client.opaque")))
	raw, err := config.GetString("shared_value")
	require.NoError(t, err)
	require.Equal(t, "synthetic-concrete-default-private", raw)
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, "synthetic-concrete-default-private")
	require.Contains(t, messages, "cfg shared_value=[REDACTED]")
	require.NotContains(t, config.AllSettings(), "client")
}

func TestDebugConfigKeepsPublicSiblingOfRootedConcreteIndex(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"rows": []any{
			map[string]any{},
			map[string]any{"client": map[string]any{"opaque": "visible-rooted-index-sibling"}},
		},
	})
	require.NoError(t, config.Option(cfg.WithConfigSetting("rows[0]", map[string]any{
		"client": debugCredentials{Opaque: "synthetic-rooted-index-private"},
	})))
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, "synthetic-rooted-index-private")
	require.Contains(t, messages, "cfg rows.0.client.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg rows.1.client.opaque=visible-rooted-index-sibling")
	require.Equal(t, "visible-rooted-index-sibling", config.AllSettings()["rows"].([]any)[1].(map[string]any)["client"].(map[string]any)["opaque"])
}

func TestDebugConfigProtectsBoxedConcreteMaps(t *testing.T) {
	for _, initial := range []bool{false, true} {
		t.Run(fmt.Sprintf("initial=%t", initial), func(t *testing.T) {
			var boxed any = map[string]any{
				"literal.client": debugCredentials{Opaque: "synthetic-boxed-private", AccessKeyID: "visible-boxed-public"},
			}
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
			if initial {
				config = cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"boxed": &boxed})
			} else {
				require.NoError(t, config.Option(cfg.WithConfigSetting("boxed", &boxed)))
			}
			var future any = map[string]debugCredentials(nil)
			require.NoError(t, config.Option(cfg.WithConfigSetting("typed", &future)))
			future = map[string]debugCredentials{
				"future.client": {Opaque: "synthetic-boxed-future-private", AccessKeyID: "visible-boxed-future-public"},
			}
			logged, messages := captureDebugConfig(t, config)
			require.NotContains(t, logged, "synthetic-boxed-private")
			require.Contains(t, messages, "cfg boxed.literal.client.opaque=[REDACTED]")
			require.Contains(t, messages, "cfg boxed.literal.client.access_key_id=visible-boxed-public")
			require.NotContains(t, logged, "synthetic-boxed-future-private")
			require.Contains(t, messages, "cfg typed.future.client.opaque=[REDACTED]")
			require.Contains(t, messages, "cfg typed.future.client.access_key_id=visible-boxed-future-public")
			require.Same(t, &boxed, config.AllSettings()["boxed"])
			require.Equal(t, "synthetic-boxed-private", boxed.(map[string]any)["literal.client"].(debugCredentials).Opaque)
		})
	}
}

func TestDebugConfigRetainedInterfaceUsesCurrentSecretTags(t *testing.T) {
	var value any
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"client": &value})
	// Initialize the logger's public defaults before comparing fingerprints.
	captureDebugConfig(t, config)
	value = debugCredentials{Opaque: "synthetic-late-secret", AccessKeyID: "visible-late-id"}

	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-late-secret")
	require.Contains(t, messages, "cfg client.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg client.access_key_id=visible-late-id")
	original, err := config.Get("client")
	require.NoError(t, err)
	require.Same(t, &value, original)
	require.Equal(t, "synthetic-late-secret", value.(debugCredentials).Opaque)

	value = debugCredentials{Opaque: "synthetic-replacement-secret", AccessKeyID: "visible-late-id"}
	replacementOutput, replacementMessages := captureDebugConfig(t, config)
	require.NotContains(t, replacementOutput, "synthetic-replacement-secret")
	require.Equal(t, messages[len(messages)-1], replacementMessages[len(replacementMessages)-1])
}

func TestDebugConfigRetainedInterfaceMasksCurrentSecretContainers(t *testing.T) {
	type containers struct {
		Map    map[string]string `cfg:"opaque_map, SeCrEt ,NoDecode"`
		Slice  []string          `cfg:"opaque_slice,secret"`
		Public string            `cfg:"public"`
	}
	var value any
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"client": &value})
	// Initialize the logger's public defaults before comparing fingerprints.
	captureDebugConfig(t, config)
	value = containers{
		Map:    map[string]string{"synthetic-private-key": "synthetic-map-secret"},
		Slice:  []string{"synthetic-slice-secret"},
		Public: "visible-container-id",
	}

	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.Contains(t, messages, "cfg client.opaque_map=[REDACTED]")
	require.Contains(t, messages, "cfg client.opaque_slice=[REDACTED]")
	require.Contains(t, messages, "cfg client.public=visible-container-id")
	require.Equal(t, "synthetic-map-secret", value.(containers).Map["synthetic-private-key"])
	require.Equal(t, []string{"synthetic-slice-secret"}, value.(containers).Slice)

	value = containers{
		Map:    map[string]string{"synthetic-new-private-key": "synthetic-new-map-secret"},
		Slice:  []string{"synthetic-new-slice-secret", "synthetic-extra-secret"},
		Public: "visible-container-id",
	}
	_, replacementMessages := captureDebugConfig(t, config)
	require.Equal(t, messages[len(messages)-1], replacementMessages[len(replacementMessages)-1])
}

func TestDebugConfigRetainedInterfacePreservesAnonymousMapping(t *testing.T) {
	type Credentials struct {
		Opaque string `cfg:" opaque , SeCrEt "`
		Public string `cfg:"public"`
	}
	type settings struct {
		Credentials `cfg:"ignored,secret"`
		Untagged    string
	}
	var value any
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"client": &value})
	value = settings{
		Credentials: Credentials{Opaque: "synthetic-embedded-secret", Public: "visible-embedded-id"},
		Untagged:    "synthetic-untagged-value",
	}

	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-")
	require.NotContains(t, output, "ignored")
	require.Contains(t, messages, "cfg client.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg client.public=visible-embedded-id")
}

func TestDebugConfigDefaultsSourceMasksWholeSecretContainer(t *testing.T) {
	type settings struct {
		Hidden map[string]string `cfg:"hidden,secret"`
		Public string            `cfg:"public"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"shared": map[string]any{"synthetic-private-key": "synthetic-private-value"},
	})
	var output settings
	require.NoError(t, config.UnmarshalKey("provider", &output,
		cfg.UnmarshalWithDefaultsFromKey("shared", "hidden")))
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, "synthetic-")
	require.Contains(t, messages, "cfg shared=[REDACTED]")
	require.Contains(t, messages, "cfg provider.hidden=[REDACTED]")
	require.Equal(t, "synthetic-private-value", output.Hidden["synthetic-private-key"])
}

func TestDebugConfigKeepsNormalizedConcreteAnnotations(t *testing.T) {
	type clients map[string]any
	type settings struct {
		Clients clients `cfg:"clients"`
	}
	type publicCredentials struct {
		Opaque string `cfg:"opaque"`
	}
	original := settings{Clients: clients{
		"private": debugCredentials{Opaque: "synthetic-normalized-private", AccessKeyID: "visible-normalized-id"},
		"public":  publicCredentials{Opaque: "visible-normalized-sibling"},
	}}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigSetting("provider", &original)))
	// The stored normalized values are independent of the original model. Its
	// later mutation must not erase annotations for those stored old values.
	original.Clients["private"] = publicCredentials{Opaque: "visible-new-model"}
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, "synthetic-normalized-private")
	require.Contains(t, messages, "cfg provider.clients.private.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg provider.clients.private.access_key_id=visible-normalized-id")
	require.Contains(t, messages, "cfg provider.clients.public.opaque=visible-normalized-sibling")
}

func TestDebugConfigRejectsUnsupportedDefaultProjection(t *testing.T) {
	type settings struct {
		Client any `cfg:"client"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"shared": "synthetic-unprojectable",
	})
	var output settings
	require.NoError(t, config.UnmarshalDefaults(&output,
		cfg.UnmarshalWithDefaultsFromKey("shared", "client.opaque")))
	logger := &debugCountingLogger{}
	err := cfg.DebugConfig(t.Context(), config, logger)
	require.ErrorContains(t, err, "client.opaque")
	require.NotContains(t, err.Error(), "synthetic-unprojectable")
	require.Empty(t, logger.messages)
}

func TestDebugConfigDefaultsForAbsentDestinationFieldDoNotAffectLoading(t *testing.T) {
	type settings struct {
		Name string `cfg:"name"`
	}
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprintf("source_present=%t", present), func(t *testing.T) {
			values := map[string]any{"component": map[string]any{"name": "visible-component"}}
			if present {
				values["shared"] = map[string]any{"repository": "visible-image"}
			}
			config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), values)
			var output settings
			require.NoError(t, config.UnmarshalKey("component", &output,
				cfg.UnmarshalWithDefaultsFromKey("shared", "image")))
			require.Equal(t, "visible-component", output.Name)
			logged, _ := captureDebugConfig(t, config)
			require.Contains(t, logged, "visible-component")
			if present {
				require.Contains(t, logged, "visible-image")
				stored, err := config.GetString("component.image.repository")
				require.NoError(t, err)
				require.Equal(t, "visible-image", stored)
			}
		})
	}
}

func TestDebugConfigPreservesPublicRawScalarSliceSettings(t *testing.T) {
	type settings struct {
		Names []string `cfg:"names"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"provider": map[string]any{"names": "visible-first,visible-second"},
	})
	var output settings
	require.NoError(t, config.UnmarshalKey("provider", &output))
	require.Equal(t, []string{"visible-first", "visible-second"}, output.Names)
	logged, messages := captureDebugConfig(t, config)
	require.Contains(t, messages, "cfg provider.names=visible-first,visible-second")
	require.Contains(t, logged, "visible-first")
}

func TestDebugConfigDoesNotExposeEncodedCompositeSliceSettings(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"clients": "synthetic-encoded-private",
	})
	var output []debugCredentials
	// Registration precedes decoding, including a failed decode of raw input.
	require.Error(t, config.UnmarshalKey("clients", &output))
	logger := &debugCountingLogger{}
	err := cfg.DebugConfig(t.Context(), config, logger)
	require.ErrorContains(t, err, "clients")
	require.NotContains(t, err.Error(), "synthetic-encoded-private")
	require.Empty(t, logger.messages)
}

func TestDebugConfigProtectsUnusedSDKSettings(t *testing.T) {
	// Importing owners supplies the metadata: no owner client, provider,
	// postprocessor, or settings reader decodes these settings before logging.
	for _, source := range []string{"map", "json", "yaml"} {
		t.Run(source, func(t *testing.T) {
			fingerprint := func(private, public string) string {
				credentials := func() map[string]any {
					return map[string]any{"access_key_id": public, "secret_access_key": private, "session_token": private}
				}
				client := func() map[string]any {
					return map[string]any{"credentials": credentials(), "region": public}
				}
				image := func() map[string]any {
					return map[string]any{"repository": public, "auth": map[string]any{"username": public, "password": private}}
				}
				values := map[string]any{
					"db": map[string]any{
						"archive":         map[string]any{"uri": map[string]any{"password": private, "host": public}},
						"report":          map[string]any{"uri": map[string]any{"password": private, "database": public}},
						"tenant.with.dot": map[string]any{"uri": map[string]any{"password": private, "user": public}},
					},
					"kafka": map[string]any{"connection": map[string]any{"unused": map[string]any{"password": private, "username": public}}},
					"cloud": map[string]any{"aws": map[string]any{
						"defaults": map[string]any{"credentials": credentials(), "clients": map[string]any{"default": client(), "unused": client()}},
						"s3":       map[string]any{"clients": map[string]any{"default": client(), "unused": client()}, "public": public},
					}},
					"api_auth_basic_users": []any{map[string]any{"username": private, "password": private}},
					"api_auth_keys":        []any{private},
					"httpserver":           map[string]any{"unused": map[string]any{"auth": map[string]any{"jwt": map[string]any{"signingSecret": private, "issuer": public}}}},
					"currency_service":     map[string]any{"providers": map[string]any{"unused": map[string]any{"api_key": private, "enabled": true}}},
					"otel":                 map[string]any{"exporter": map[string]any{"headers": map[string]any{private: private}, "host": public}},
					"test": map[string]any{
						"defaults":          map[string]any{"images": map[string]any{"mysql": image(), "unused": image()}},
						"container_manager": map[string]any{"runner": map[string]any{"local": map[string]any{"auth": map[string]any{"username": public, "password": private}}}},
						"components": map[string]any{"mysql": map[string]any{"unused": map[string]any{
							"credentials": map[string]any{"user_password": private, "root_password": private},
							"image":       image(),
						}}},
					},
					"unowned": map[string]any{"password": public},
				}
				config := unusedSDKDebugConfig(t, source, values)
				originalSettings := config.AllSettings()
				output, messages := captureDebugConfig(t, config)
				require.NotContains(t, output, private)
				for _, path := range []string{
					"db.archive.uri.password", "db.report.uri.password", "db.tenant.with.dot.uri.password",
					"kafka.connection.unused.password",
					"cloud.aws.defaults.credentials.secret_access_key", "cloud.aws.defaults.credentials.session_token",
					"cloud.aws.defaults.clients.default.credentials.secret_access_key", "cloud.aws.defaults.clients.unused.credentials.session_token",
					"cloud.aws.s3.clients.default.credentials.secret_access_key", "cloud.aws.s3.clients.unused.credentials.session_token",
					"api_auth_basic_users", "api_auth_keys", "httpserver.unused.auth.jwt.signingSecret",
					"currency_service.providers.unused.api_key", "otel.exporter.headers",
					"test.defaults.images.mysql.auth.password", "test.defaults.images.unused.auth.password",
					"test.container_manager.runner.local.auth.password",
					"test.components.mysql.unused.credentials.user_password", "test.components.mysql.unused.credentials.root_password",
					"test.components.mysql.unused.image.auth.password",
				} {
					require.Contains(t, messages, "cfg "+path+"=[REDACTED]")
				}
				for _, path := range []string{
					"db.archive.uri.host", "db.report.uri.database", "kafka.connection.unused.username",
					"cloud.aws.defaults.credentials.access_key_id", "cloud.aws.s3.clients.unused.region",
					"cloud.aws.s3.public", "httpserver.unused.auth.jwt.issuer", "otel.exporter.host",
					"test.defaults.images.mysql.repository", "test.components.mysql.unused.image.auth.username", "unowned.password",
				} {
					require.Contains(t, messages, "cfg "+path+"="+public)
				}
				require.Contains(t, messages, "cfg currency_service.providers.unused.enabled=true")
				storedSettings := config.AllSettings()
				for root := range values {
					require.Equal(t, originalSettings[root], storedSettings[root], root)
				}
				require.Equal(t, private, values["db"].(map[string]any)["archive"].(map[string]any)["uri"].(map[string]any)["password"])
				stored, err := config.GetString("db.archive.uri.password")
				require.NoError(t, err)
				require.Equal(t, private, stored)

				return debugConfigFingerprint(t, messages)
			}
			baseline := fingerprint("synthetic-unused-before", "visible-unused")
			require.Equal(t, baseline, fingerprint("synthetic-unused-after", "visible-unused"))
			require.NotEqual(t, baseline, fingerprint("synthetic-unused-before", "visible-changed"))
		})
	}
}

func unusedSDKDebugConfig(t *testing.T, source string, values map[string]any) cfg.Config {
	t.Helper()
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	if source == "map" {
		require.NoError(t, config.Option(cfg.WithConfigMap(values)))
	} else {
		// JSON is also valid YAML and exercises the YAML source decoder.
		data, err := json.Marshal(values)
		require.NoError(t, err)
		require.NoError(t, config.Option(cfg.WithConfigBytes(data, source)))
	}

	return config
}

func debugConfigFingerprint(t *testing.T, messages []string) string {
	t.Helper()
	for _, message := range messages {
		if strings.HasPrefix(message, "cfg fingerprint: ") {
			return message
		}
	}
	t.Fatal("missing fingerprint")

	return ""
}

func TestRegisterConfigDebugTypeAppliesToExistingConfig(t *testing.T) {
	root := "owner_debug_registration"
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		root: map[string]any{"unused": map[string]any{"opaque": "synthetic-owner-private", "access_key_id": "visible-owner"}},
	})
	// The unique root and stable type make repeat/full-suite runs independent
	// without resetting or mutating registrations owned by imported packages.
	typ := reflect.TypeFor[map[string]debugCredentials]()
	cfg.RegisterConfigDebugType(root, nil)
	cfg.RegisterConfigDebugType(root, typ)
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 8 {
				cfg.RegisterConfigDebugType("."+root+".", typ)
			}
		}()
	}
	logger := &debugCountingLogger{}
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))
	messages := logger.messages
	require.NotContains(t, strings.Join(messages, "\n"), "synthetic-owner-private")
	require.Contains(t, messages, "cfg "+root+".unused.opaque=[REDACTED]")
	require.Contains(t, messages, "cfg "+root+".unused.access_key_id=visible-owner")
	workers.Wait()
	repeated := &debugCountingLogger{}
	require.NoError(t, cfg.DebugConfig(t.Context(), config, repeated))
	require.Equal(t, messages, repeated.messages)
}

func TestDebugConfigUnusedSDKUnsupportedShapeFailsBeforeLogging(t *testing.T) {
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"db": map[string]any{"archive": map[string]any{"uri": "synthetic-encoded-credentials"}},
	})
	// Metadata does not reject source loading or perform operational decoding.
	stored, err := config.GetString("db.archive.uri")
	require.NoError(t, err)
	require.Equal(t, "synthetic-encoded-credentials", stored)
	logger := &debugCountingLogger{}
	err = cfg.DebugConfig(t.Context(), config, logger)
	require.ErrorContains(t, err, "db.archive.uri")
	require.NotContains(t, err.Error(), "synthetic-encoded-credentials")
	require.Empty(t, logger.messages)
}

func TestDebugConfigProtectsUnusedLiteralDatabaseNames(t *testing.T) {
	fingerprint := func(private string) string {
		// A raw constructor map retains literal dots; source merges interpret
		// unescaped dots as selectors rather than collection member names.
		values := map[string]any{"db": map[string]any{
			"tenant.with.dot": map[string]any{"uri": map[string]any{"password": private, "host": "visible-literal-host"}},
		}}
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), values)
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, private)
		require.Contains(t, messages, "cfg db.tenant.with.dot.uri.password=[REDACTED]")
		require.Contains(t, messages, "cfg db.tenant.with.dot.uri.host=visible-literal-host")
		require.Equal(t, values["db"], config.AllSettings()["db"])

		return debugConfigFingerprint(t, messages)
	}
	require.Equal(t, fingerprint("synthetic-literal-before"), fingerprint("synthetic-literal-after"))
}

func TestDebugConfigRejectsAmbiguousUnusedDatabaseNamesBeforeLogging(t *testing.T) {
	config := unusedSDKDebugConfig(t, "map", map[string]any{
		"db": map[string]any{"tenant.uri.dot": map[string]any{
			"uri": map[string]any{"password": "synthetic-ambiguous-password", "host": "visible-ambiguous-host"},
		}},
	})
	logger := &debugCountingLogger{}
	err := cfg.DebugConfig(t.Context(), config, logger)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-ambiguous-password")
	require.Empty(t, logger.messages)
}

func TestDebugConfigCollectionStopsAtPublicModelFields(t *testing.T) {
	config := unusedSDKDebugConfig(t, "map", map[string]any{
		"db": map[string]any{"archive": map[string]any{
			"uri":        map[string]any{"password": "synthetic-boundary-password", "host": "visible-boundary-host"},
			"parameters": map[string]any{"driver": "visible-public-driver", "uri": "visible-public-uri"},
			"metadata":   map[string]any{"password": "visible-unowned-password"},
		}},
	})
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, "synthetic-boundary-password")
	require.Contains(t, messages, "cfg db.archive.uri.password=[REDACTED]")
	require.Contains(t, messages, "cfg db.archive.uri.host=visible-boundary-host")
	require.Contains(t, messages, "cfg db.archive.parameters.driver=visible-public-driver")
	require.Contains(t, messages, "cfg db.archive.parameters.uri=visible-public-uri")
	require.Contains(t, messages, "cfg db.archive.metadata.password=visible-unowned-password")
}

func TestDebugConfigCollectionDoesNotInspectMaskedContainers(t *testing.T) {
	type settings struct {
		Opaque any    `cfg:"opaque,secret"`
		Public string `cfg:"public"`
	}
	type cyclicSecret struct {
		Next  *cyclicSecret
		Value string
	}
	cyclic := &cyclicSecret{Value: "synthetic-masked-cycle"}
	cyclic.Next = cyclic
	root := "owner_debug_masked_containers"
	cfg.RegisterConfigDebugType(root, reflect.TypeFor[map[string]settings]())
	for _, secret := range []any{
		map[debugFormattingKey]string{{}: "synthetic-masked-unsupported"},
		cyclic,
	} {
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
			root: map[string]any{"unused": map[string]any{"opaque": secret, "public": "visible-masked-sibling"}},
		})
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, "synthetic-masked")
		require.Contains(t, messages, "cfg "+root+".unused.opaque=[REDACTED]")
		require.Contains(t, messages, "cfg "+root+".unused.public=visible-masked-sibling")
		require.Same(t, cyclic, cyclic.Next)
	}
}

func TestDebugConfigRejectedNestedDatabaseCandidateFailsClosed(t *testing.T) {
	config := unusedSDKDebugConfig(t, "map", map[string]any{
		"db": map[string]any{"tenant.uri.dot": map[string]any{
			// Migrations precedes Uri in the original SDK model. Its invalid
			// public shape must not let probing stop before the secret tag.
			"migrations": "visible-rejected-public-shape",
			"uri":        map[string]any{"password": "synthetic-rejected-candidate", "host": "visible-rejected-host"},
		}},
	})
	stored, err := config.GetString("db.tenant.uri.dot.uri.password")
	require.NoError(t, err)
	require.Equal(t, "synthetic-rejected-candidate", stored)
	logger := &debugCountingLogger{}
	err = cfg.DebugConfig(t.Context(), config, logger)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-rejected-candidate")
	require.Empty(t, logger.messages)
}

func TestDebugConfigUnusedOwnerNamespacesKeepPublicLeaves(t *testing.T) {
	for _, source := range []string{"map", "json", "yaml"} {
		t.Run(source, func(t *testing.T) {
			config := unusedSDKDebugConfig(t, source, map[string]any{
				"cloud": map[string]any{"aws": map[string]any{"defaults": map[string]any{
					"region":      "visible-shared-region",
					"credentials": map[string]any{"access_key_id": "visible-shared-access", "secret_access_key": "synthetic-shared-only"},
				}}},
				"httpserver": map[string]any{"api": map[string]any{"port": 8080}},
			})
			output, messages := captureDebugConfig(t, config)
			require.NotContains(t, output, "synthetic-shared-only")
			require.Contains(t, messages, "cfg cloud.aws.defaults.region=visible-shared-region")
			require.Contains(t, messages, "cfg cloud.aws.defaults.credentials.access_key_id=visible-shared-access")
			require.Contains(t, messages, "cfg cloud.aws.defaults.credentials.secret_access_key=[REDACTED]")
			require.Contains(t, messages, "cfg httpserver.api.port=8080")
			require.NotEmpty(t, debugConfigFingerprint(t, messages))
		})
	}
}

func TestDebugConfigLiteralBracketsDoNotOverwriteIndexes(t *testing.T) {
	values := map[string]any{
		"x[0]":  "visible-literal-brackets",
		"x":     []any{"visible-sequence-index"},
		`x\[0]`: "visible-backslash-brackets",
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), values)
	logger := &debugCountingLogger{}
	require.NoError(t, cfg.DebugConfig(t.Context(), config, logger))
	require.Contains(t, logger.messages, "cfg x[0]=visible-literal-brackets")
	require.Contains(t, logger.messages, "cfg x.0=visible-sequence-index")
	require.Contains(t, logger.messages, `cfg x\[0]=visible-backslash-brackets`)
	require.Equal(t, values, config.AllSettings())
	for range 8 {
		repeated := &debugCountingLogger{}
		require.NoError(t, cfg.DebugConfig(t.Context(), config, repeated))
		require.Equal(t, logger.messages, repeated.messages)
	}
}

func TestDebugConfigBracketMasksStayScopedToLiteralKeys(t *testing.T) {
	type publicCredentials struct {
		Opaque string `cfg:"opaque"`
	}
	type settings struct {
		Clients map[string]any `cfg:"clients"`
	}
	for _, source := range []string{"retained", "retained_interface"} {
		t.Run(source, func(t *testing.T) {
			fingerprint := func(private, public string) string {
				native := &settings{Clients: map[string]any{
					"x[0]":  &debugCredentials{Opaque: private, AccessKeyID: "visible-private-sibling"},
					"x":     []any{&publicCredentials{Opaque: public}},
					`x\[0]`: &publicCredentials{Opaque: "visible-backslash-sibling"},
				}}
				var carrier any = native
				if source == "retained_interface" {
					boxed := carrier
					carrier = &boxed
				}
				config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"provider": carrier})
				output, messages := captureDebugConfig(t, config)
				require.NotContains(t, output, private)
				require.Contains(t, messages, "cfg provider.clients.x[0].opaque=[REDACTED]")
				require.Contains(t, messages, "cfg provider.clients.x[0].access_key_id=visible-private-sibling")
				require.Contains(t, messages, "cfg provider.clients.x.0.opaque="+public)
				require.Contains(t, messages, `cfg provider.clients.x\[0].opaque=visible-backslash-sibling`)
				require.Equal(t, private, native.Clients["x[0]"].(*debugCredentials).Opaque)
				require.Equal(t, public, native.Clients["x"].([]any)[0].(*publicCredentials).Opaque)

				return debugConfigFingerprint(t, messages)
			}
			baseline := fingerprint("synthetic-literal-bracket-before", "visible-index-before")
			require.Equal(t, baseline, fingerprint("synthetic-literal-bracket-after", "visible-index-before"))
			require.NotEqual(t, baseline, fingerprint("synthetic-literal-bracket-before", "visible-index-after"))
		})
	}
}

func TestDebugConfigProjectedIndexMasksDoNotHideLiteralBracketKeys(t *testing.T) {
	type settings struct {
		Rows []debugCredentials `cfg:"rows"`
	}
	root := "owner_debug_bracket_projection"
	cfg.RegisterConfigDebugType(root, reflect.TypeFor[settings]())
	fingerprint := func(private string) string {
		values := map[string]any{root: map[string]any{
			"rows":    []any{map[string]any{"opaque": private, "access_key_id": "visible-projected-sibling"}},
			"rows[0]": map[string]any{"opaque": "visible-literal-opaque"},
		}}
		config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), values)
		output, messages := captureDebugConfig(t, config)
		require.NotContains(t, output, private)
		require.Contains(t, messages, "cfg "+root+".rows.0.opaque=[REDACTED]")
		require.Contains(t, messages, "cfg "+root+".rows.0.access_key_id=visible-projected-sibling")
		require.Contains(t, messages, "cfg "+root+".rows[0].opaque=visible-literal-opaque")
		require.Equal(t, values[root], config.AllSettings()[root])

		return debugConfigFingerprint(t, messages)
	}
	require.Equal(t, fingerprint("synthetic-projected-before"), fingerprint("synthetic-projected-after"))
}

func TestDebugConfigNormalizedSelectorKeysKeepConcreteAnnotations(t *testing.T) {
	for _, entry := range []struct {
		name     string
		key      string
		selector string
		display  string
		grouped  bool
	}{
		{name: "bracket_member", key: "x[0]", selector: "clients.x[0]", display: "clients.x.0"},
		{name: "dotted_member", key: "tenant.one", selector: "clients.tenant.one", display: "clients.tenant.one"},
		{name: "dotted_field", key: "private", selector: "clients.group.private", display: "clients.group.private", grouped: true},
	} {
		t.Run(entry.name, func(t *testing.T) {
			fingerprint := func(private, public string) string {
				return normalizedSelectorDebugFingerprint(t, entry.key, entry.selector, entry.display, entry.grouped, private, public)
			}
			baseline := fingerprint("synthetic-normalized-selector-before", "visible-normalized-selector")
			require.Equal(t, baseline, fingerprint("synthetic-normalized-selector-after", "visible-normalized-selector"))
			require.NotEqual(t, baseline, fingerprint("synthetic-normalized-selector-before", "visible-normalized-selector-changed"))
		})
	}
}

func normalizedSelectorDebugFingerprint(t *testing.T, key, selector, display string, grouped bool, private, public string) string {
	t.Helper()
	type clients map[string]any
	type settings struct {
		Clients clients `cfg:"clients"`
	}
	type groupedSettings struct {
		Clients clients `cfg:"clients.group"`
	}
	type publicCredentials struct {
		Opaque string `cfg:"opaque"`
	}
	native := clients{
		key:      debugCredentials{Opaque: private, AccessKeyID: public},
		"public": publicCredentials{Opaque: "visible-normalized-public-sibling"},
	}
	var carrier any = &settings{Clients: native}
	publicPath := "provider.clients.public.opaque"
	if grouped {
		carrier = &groupedSettings{Clients: native}
		publicPath = "provider.clients.group.public.opaque"
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider())
	require.NoError(t, config.Option(cfg.WithConfigSetting("provider", carrier)))
	original := config.AllSettings()["provider"]
	require.Equal(t, private, native[key].(debugCredentials).Opaque)
	// Concrete models were normalized into independent settings. Replacing the
	// retained source model must not erase the stored old value's annotations.
	native[key] = publicCredentials{Opaque: "visible-replaced-source-model"}
	output, messages := captureDebugConfig(t, config)
	require.NotContains(t, output, private)
	require.Contains(t, messages, "cfg provider."+display+".opaque=[REDACTED]")
	require.Contains(t, messages, "cfg provider."+display+".access_key_id="+public)
	require.Contains(t, messages, "cfg "+publicPath+"=visible-normalized-public-sibling")
	require.Equal(t, original, config.AllSettings()["provider"])
	stored, err := config.GetString("provider." + selector + ".opaque")
	require.NoError(t, err)
	require.Equal(t, private, stored)
	require.Equal(t, publicCredentials{Opaque: "visible-replaced-source-model"}, native[key])

	return debugConfigFingerprint(t, messages)
}

func TestDebugConfigDefaultsResolveCollectionDestinations(t *testing.T) {
	type sliceSettings struct {
		Rows []gosoAws.Credentials `cfg:"rows"`
	}
	type mapSettings struct {
		Clients map[string]gosoAws.Credentials `cfg:"clients"`
	}
	type groupedSettings struct {
		Clients map[string]gosoAws.Credentials `cfg:"clients.group"`
	}
	type indexedSettings struct {
		First gosoAws.Credentials `cfg:"rows[0]"`
	}
	type interfaceSettings struct {
		Clients map[string]any `cfg:"clients"`
	}
	for _, entry := range []struct {
		name      string
		target    string
		display   string
		newOutput func() any
	}{
		{name: "empty_slice", target: "rows[0]", display: "rows.0", newOutput: func() any { return &sliceSettings{} }},
		{name: "empty_named_map", target: "clients.default", display: "clients.default", newOutput: func() any { return &mapSettings{} }},
		{name: "escaped_named_map", target: `clients.tenant\.one`, display: `clients.tenant\.one`, newOutput: func() any { return &mapSettings{} }},
		{name: "dotted_field", target: "clients.group.default", display: "clients.group.default", newOutput: func() any { return &groupedSettings{} }},
		{name: "indexed_field", target: "rows[0]", display: "rows.0", newOutput: func() any { return &indexedSettings{} }},
		{name: "present_interface_pointer", target: "clients.default", display: "clients.default", newOutput: func() any {
			return &interfaceSettings{Clients: map[string]any{"default": &gosoAws.Credentials{}}}
		}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			fingerprint := func(private, public string) string {
				return defaultCollectionDebugFingerprint(t, entry.newOutput(), entry.target, entry.display, private, public)
			}
			baseline := fingerprint("synthetic-collection-default-before", "visible-collection-default")
			require.Equal(t, baseline, fingerprint("synthetic-collection-default-after", "visible-collection-default"))
			require.NotEqual(t, baseline, fingerprint("synthetic-collection-default-before", "visible-collection-default-changed"))
		})
	}
}

func defaultCollectionDebugFingerprint(t *testing.T, output any, target, display, private, public string) string {
	t.Helper()
	shared := map[string]any{"secret_access_key": private, "session_token": private, "access_key_id": public}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{"shared": shared})
	require.NoError(t, config.UnmarshalKey("provider", output, cfg.UnmarshalWithDefaultsFromKey("shared", target)))
	destination := config.AllSettings()["provider"]
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, private)
	for _, prefix := range []string{"shared", "provider." + display} {
		require.Contains(t, messages, "cfg "+prefix+".secret_access_key=[REDACTED]")
		require.Contains(t, messages, "cfg "+prefix+".session_token=[REDACTED]")
		require.Contains(t, messages, "cfg "+prefix+".access_key_id="+public)
	}
	require.Equal(t, shared, config.AllSettings()["shared"])
	require.Equal(t, destination, config.AllSettings()["provider"])
	for _, path := range []string{"shared.secret_access_key", "provider." + target + ".secret_access_key"} {
		stored, err := config.GetString(path)
		require.NoError(t, err)
		require.Equal(t, private, stored)
	}

	return debugConfigFingerprint(t, messages)
}

func TestDebugConfigDefaultsCollectionSecretAncestorCollapsesSource(t *testing.T) {
	type settings struct {
		Clients map[string]gosoAws.Credentials `cfg:"clients,secret"`
		Public  string                         `cfg:"public"`
	}
	shared := map[string]any{
		"secret_access_key": "synthetic-secret-ancestor",
		"access_key_id":     "synthetic-secret-container-key",
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"shared":   shared,
		"provider": map[string]any{"public": "visible-secret-ancestor-sibling"},
	})
	var output settings
	require.NoError(t, config.UnmarshalKey("provider", &output, cfg.UnmarshalWithDefaultsFromKey("shared", "clients.default")))
	logged, messages := captureDebugConfig(t, config)
	require.NotContains(t, logged, "synthetic-secret")
	require.Contains(t, messages, "cfg shared=[REDACTED]")
	require.Contains(t, messages, "cfg provider.clients=[REDACTED]")
	require.Contains(t, messages, "cfg provider.public=visible-secret-ancestor-sibling")
	require.Equal(t, "synthetic-secret-ancestor", output.Clients["default"].SecretAccessKey)
	require.Equal(t, shared, config.AllSettings()["shared"])
}

func TestDebugConfigDefaultsUnresolvedCollectionInterfaceIsLoggingOnly(t *testing.T) {
	type settings struct {
		Clients map[string]any `cfg:"clients"`
	}
	config := cfg.NewWithInterfaces(cfg.NewMemoryEnvProvider(), map[string]any{
		"shared": map[string]any{"secret_access_key": "synthetic-unresolved-collection"},
	})
	var output settings
	require.NoError(t, config.UnmarshalKey("provider", &output, cfg.UnmarshalWithDefaultsFromKey("shared", "clients.default")))
	logger := &debugCountingLogger{}
	err := cfg.DebugConfig(t.Context(), config, logger)
	require.ErrorContains(t, err, "clients.default")
	require.NotContains(t, err.Error(), "synthetic-unresolved-collection")
	require.Empty(t, logger.messages)
	stored, err := config.GetString("provider.clients.default.secret_access_key")
	require.NoError(t, err)
	require.Equal(t, "synthetic-unresolved-collection", stored)
}
