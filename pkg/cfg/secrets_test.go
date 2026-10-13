package cfg_test

import (
	"fmt"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
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
