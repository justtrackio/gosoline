package mdlsub

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSubscriberFilterPolicy_Empty(t *testing.T) {
	attrs := buildSubscriberFilterPolicy(nil)
	assert.Nil(t, attrs)

	attrs = buildSubscriberFilterPolicy([]string{})
	assert.Nil(t, attrs)
}

func TestBuildSubscriberFilterPolicy_Single(t *testing.T) {
	attrs := buildSubscriberFilterPolicy([]string{"mcoins.marketing.management.app"})
	assert.Equal(t, map[string][]string{
		"modelId": {"mcoins.marketing.management.app"},
	}, attrs)
}

func TestBuildSubscriberFilterPolicy_Multiple(t *testing.T) {
	attrs := buildSubscriberFilterPolicy([]string{
		"mcoins.marketing.management.app",
		"mcoins.marketing.management.network",
		"mcoins.marketing.management.platform",
	})
	assert.Equal(t, map[string][]string{
		"modelId": {
			"mcoins.marketing.management.app",
			"mcoins.marketing.management.network",
			"mcoins.marketing.management.platform",
		},
	}, attrs)
}

func TestCollectModelIdsByInput(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "subscriber",
			"env":  "dev",
			"tags": map[string]any{
				"project": "mcoins",
				"family":  "marketing",
				"group":   "attribution",
			},
		},
	})

	settings := &Settings{
		Subscribers: map[string]*SubscriberSettings{
			"app": {
				Input: "sns",
				SourceModel: SubscriberModel{
					ModelId: mdl.ModelId{
						Name: "app",
						Tags: map[string]string{
							"project": "mcoins",
							"family":  "marketing",
							"group":   "management",
						},
						DomainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
					},
					Shared: true,
				},
			},
			"network": {
				Input: "sns",
				SourceModel: SubscriberModel{
					ModelId: mdl.ModelId{
						Name: "network",
						Tags: map[string]string{
							"project": "mcoins",
							"family":  "marketing",
							"group":   "management",
						},
						DomainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
					},
					Shared: true,
				},
			},
			"kafkaModel": {
				Input: "kafka",
				SourceModel: SubscriberModel{
					ModelId: mdl.ModelId{
						Name: "kafkaModel",
					},
				},
			},
		},
	}

	result, err := collectModelIdsByInput(config, settings)
	require.NoError(t, err)

	// kafkaModel should not appear since it's not SNS input
	// Both app and network share the same input key because they are shared
	assert.Len(t, result, 1)

	for _, modelIds := range result {
		assert.Contains(t, modelIds, "mcoins.marketing.management.app")
		assert.Contains(t, modelIds, "mcoins.marketing.management.network")
		assert.Len(t, modelIds, 2)
	}
}

func TestCollectModelIdsByInput_NonShared(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "subscriber",
			"env":  "dev",
			"tags": map[string]any{
				"project": "mcoins",
				"family":  "marketing",
				"group":   "attribution",
			},
		},
	})

	settings := &Settings{
		Subscribers: map[string]*SubscriberSettings{
			"app": {
				Input: "sns",
				SourceModel: SubscriberModel{
					ModelId: mdl.ModelId{
						Name: "app",
						Tags: map[string]string{
							"project": "mcoins",
							"family":  "marketing",
							"group":   "management",
						},
						DomainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
					},
				},
			},
			"network": {
				Input: "sns",
				SourceModel: SubscriberModel{
					ModelId: mdl.ModelId{
						Name: "network",
						Tags: map[string]string{
							"project": "mcoins",
							"family":  "marketing",
							"group":   "management",
						},
						DomainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
					},
				},
			},
		},
	}

	result, err := collectModelIdsByInput(config, settings)
	require.NoError(t, err)

	// Non-shared subscribers get separate input keys
	assert.Len(t, result, 2)

	for _, modelIds := range result {
		assert.Len(t, modelIds, 1)
	}
}

func TestSnsSubscriberInputConfigPostProcessor_SetsFilterAttributes(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"name": "subscriber",
			"tags": map[string]any{
				"project": "mcoins",
				"family":  "marketing",
				"group":   "attribution",
			},
		},
	})

	subscriberSettings := &SubscriberSettings{
		Input: "sns",
		SourceModel: SubscriberModel{
			ModelId: mdl.ModelId{
				Name: "app",
				Tags: map[string]string{
					"project": "mcoins",
					"family":  "marketing",
					"group":   "management",
				},
				DomainPattern: "{app.tags.project}.{app.tags.family}.{app.tags.group}",
			},
		},
	}

	modelIds := []string{
		"mcoins.marketing.management.app",
		"mcoins.marketing.management.network",
	}

	option, err := snsSubscriberInputConfigPostProcessor(config, "app", subscriberSettings, modelIds)
	require.NoError(t, err)
	require.NotNil(t, option)

	err = config.Option(option)
	require.NoError(t, err)

	// Read back the input config to verify attributes were set
	inputKey := stream.ConfigurableInputKey("subscriber-app")

	inputSettings := &stream.SnsInputConfiguration{}
	err = config.UnmarshalKey(inputKey, inputSettings)
	require.NoError(t, err)

	require.Len(t, inputSettings.Targets, 1)
	assert.Equal(t, map[string][]string{
		"modelId": {"mcoins.marketing.management.app", "mcoins.marketing.management.network"},
	}, inputSettings.Targets[0].FilterPolicy)
}
