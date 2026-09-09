package kvstore

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// configMapConfigurableItem is a small value used to round-trip through the
// ConfigMap backed KvStore in the configurable wiring tests.
type configMapConfigurableItem struct {
	Name string `json:"name"`
	Age  uint   `json:"age"`
}

// overrideKubernetesClient replaces the kubernetes client factory NewConfigMapKvStore
// uses and returns a restore function. It returns the given client for every
// request, so the factory never touches the real cluster or kubeconfig.
func overrideKubernetesClient(t *testing.T, client kubernetes.Interface) {
	t.Helper()

	original := newKubernetesClient
	newKubernetesClient = func(_ context.Context, _ cfg.Config, _ log.Logger, _ string) (kubernetes.Interface, error) {
		return client, nil
	}

	t.Cleanup(func() {
		newKubernetesClient = original
	})
}

func TestNewConfigMapKvStore_Config(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)

	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"mystore": map[string]any{
				"configmap": map[string]any{
					"namespace": "test-ns",
					"compression": map[string]any{
						"enabled": true,
						"algo":    ConfigMapCompressionGzip,
						"level":   9,
					},
					"encoding": map[string]any{
						"enabled": true,
						"format":  ConfigMapEncodingBase64,
					},
				},
			},
		},
	})

	store, err := NewConfigMapKvStore[configMapConfigurableItem](t.Context(), config, log.NewLogger(), &Settings{
		ModelId: mdl.ModelId{Name: "mystore"},
	})
	require.NoError(t, err)

	// the store is usable and round-trips a value
	require.NoError(t, store.Put(t.Context(), "foo", configMapConfigurableItem{Name: "bar", Age: 42}))

	var got configMapConfigurableItem
	found, err := store.Get(t.Context(), "foo", &got)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, configMapConfigurableItem{Name: "bar", Age: 42}, got)

	// the value is stored in the key's dedicated configmap (named
	// kvstore-<storeName>-<keyName>) under the key name, compressed and
	// base64 encoded
	cm, err := client.CoreV1().ConfigMaps("test-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, cm.Data, "foo")
	assert.NotEqual(t, `{"name":"bar","age":42}`, cm.Data["foo"])
}

func TestNewConfigMapKvStore_ConfigDefaults(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)
	overrideNamespaceResolver(t, "resolved-ns")

	// no configmap section at all: the store uses the namespace the
	// application currently runs in (faked to "resolved-ns"), no compression
	// or encoding
	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"mystore": map[string]any{},
		},
	})

	store, err := NewConfigMapKvStore[configMapConfigurableItem](appctx.WithContainer(t.Context()), config, log.NewLogger(), &Settings{
		ModelId: mdl.ModelId{Name: "mystore"},
	})
	require.NoError(t, err)

	require.NoError(t, store.Put(t.Context(), "foo", configMapConfigurableItem{Name: "bar", Age: 1}))

	cm, err := client.CoreV1().ConfigMaps("resolved-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, `{"name":"bar","age":1}`, cm.Data["foo"])
}

func TestConfigMapConfiguration_Unmarshal(t *testing.T) {
	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"mystore": map[string]any{
				"configmap": map[string]any{
					"namespace": "custom-ns",
					"encoding": map[string]any{
						"enabled": true,
						"format":  ConfigMapEncodingJSON,
					},
				},
			},
		},
	})

	configuration := ConfigMapConfiguration{}
	require.NoError(t, config.UnmarshalKey("kvstore.mystore.configmap", &configuration))
	assert.Equal(t, "custom-ns", configuration.Namespace)
	assert.True(t, configuration.Encoding.Enabled)
	assert.Equal(t, ConfigMapEncodingJSON, configuration.Encoding.Format)
	assert.False(t, configuration.Compression.Enabled)
}

func TestNewConfigurableKvStore_ChainWithConfigMapElement(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)

	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "app",
			"model_id": map[string]any{
				"domain_pattern": "test",
			},
		},
		"kvstore": map[string]any{
			"mystore": map[string]any{
				"type":     TypeChain,
				"elements": []any{TypeConfigMap},
				"configmap": map[string]any{
					"namespace": "test-ns",
				},
			},
		},
	})

	store, err := NewConfigurableKvStore[configMapConfigurableItem](t.Context(), config, log.NewLogger(), "mystore")
	require.NoError(t, err)

	require.NoError(t, store.Put(t.Context(), "foo", configMapConfigurableItem{Name: "bar", Age: 7}))

	var got configMapConfigurableItem
	found, err := store.Get(t.Context(), "foo", &got)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, configMapConfigurableItem{Name: "bar", Age: 7}, got)

	// the chain wrote through to the key's dedicated configmap
	cm, err := client.CoreV1().ConfigMaps("test-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, cm.Data, "foo")
}
