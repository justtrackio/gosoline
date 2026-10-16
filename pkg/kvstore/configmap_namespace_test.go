package kvstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// overrideNamespaceResolver replaces the namespace resolver NewConfigMapKvStore
// uses and returns a restore function. It makes the namespace resolution
// deterministic so the tests never depend on the namespace of the machine
// they run on (service account file or kubeconfig).
func overrideNamespaceResolver(t *testing.T, namespace string) {
	t.Helper()

	original := namespaceResolver
	namespaceResolver = func() (string, error) {
		return namespace, nil
	}

	t.Cleanup(func() {
		namespaceResolver = original
	})
}

// writeServiceAccountNamespaceFile writes the given content to a temporary
// file that replaces the mounted service account namespace file and restores
// the original path in a cleanup.
func writeServiceAccountNamespaceFile(t *testing.T, content string) {
	t.Helper()

	file := filepath.Join(t.TempDir(), "namespace")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

	original := serviceAccountNamespaceFile
	serviceAccountNamespaceFile = file

	t.Cleanup(func() {
		serviceAccountNamespaceFile = original
	})
}

// hideNamespaceSources points the service account namespace file at a path
// that does not exist and KUBECONFIG at a kubeconfig without any usable
// context, so resolveNamespaceFromFilesystem has no namespace source.
func hideNamespaceSources(t *testing.T) {
	t.Helper()

	original := serviceAccountNamespaceFile
	serviceAccountNamespaceFile = filepath.Join(t.TempDir(), "does-not-exist")
	t.Cleanup(func() { serviceAccountNamespaceFile = original })

	kubeconfigFile := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfigFile, []byte("apiVersion: v1\nkind: Config\n"), 0o600))
	t.Setenv("KUBECONFIG", kubeconfigFile)
}

func TestConfigMapKvStore_NamespaceResolvedFromServiceAccount(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)
	// no explicit namespace in the config: the namespace file of the mounted
	// service account (the same mount the in cluster client authenticates
	// from) decides
	writeServiceAccountNamespaceFile(t, "pod-ns\n")

	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"mystore": map[string]any{},
		},
	})

	ctx := appctx.WithContainer(t.Context())
	store, err := NewConfigMapKvStore[configMapConfigurableItem](ctx, config, log.NewLogger(), &Settings{
		ModelId: mdl.ModelId{Name: "mystore"},
	})
	require.NoError(t, err)

	require.NoError(t, store.Put(t.Context(), "foo", configMapConfigurableItem{Name: "bar", Age: 1}))

	cm, err := client.CoreV1().ConfigMaps("pod-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, `{"name":"bar","age":1}`, cm.Data["foo"])
}

func TestConfigMapKvStore_NamespaceConfiguredOverridesResolved(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)
	// both sources are present: the configured namespace must win over the
	// service account namespace
	writeServiceAccountNamespaceFile(t, "pod-ns\n")

	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"mystore": map[string]any{
				"configmap": map[string]any{
					"namespace": "other-ns",
				},
			},
		},
	})

	ctx := appctx.WithContainer(t.Context())
	store, err := NewConfigMapKvStore[configMapConfigurableItem](ctx, config, log.NewLogger(), &Settings{
		ModelId: mdl.ModelId{Name: "mystore"},
	})
	require.NoError(t, err)

	require.NoError(t, store.Put(t.Context(), "foo", configMapConfigurableItem{Name: "bar", Age: 2}))

	cm, err := client.CoreV1().ConfigMaps("other-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, `{"name":"bar","age":2}`, cm.Data["foo"])

	// nothing was written to the resolved namespace
	_, err = client.CoreV1().ConfigMaps("pod-ns").Get(t.Context(), "kvstore-mystore-foo", metav1.GetOptions{})
	require.Error(t, err)
}

func TestResolveNamespaceFromFilesystem(t *testing.T) {
	t.Run("service account namespace file wins", func(t *testing.T) {
		writeServiceAccountNamespaceFile(t, "  sa-ns \n")

		namespace, err := resolveNamespaceFromFilesystem()
		require.NoError(t, err)
		assert.Equal(t, "sa-ns", namespace)
	})

	t.Run("kubeconfig context namespace is the fallback", func(t *testing.T) {
		// point the service account file at a path that does not exist so the
		// resolver falls back to the kubeconfig loading rules
		original := serviceAccountNamespaceFile
		serviceAccountNamespaceFile = filepath.Join(t.TempDir(), "does-not-exist")
		t.Cleanup(func() { serviceAccountNamespaceFile = original })

		kubeconfig := `apiVersion: v1
kind: Config
current-context: ctx-a
contexts:
- name: ctx-a
  context:
    namespace: kube-ns
    cluster: c
    user: u
clusters:
- name: c
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u
  user: {}
`
		kubeconfigFile := filepath.Join(t.TempDir(), "kubeconfig")
		require.NoError(t, os.WriteFile(kubeconfigFile, []byte(kubeconfig), 0o600))
		t.Setenv("KUBECONFIG", kubeconfigFile)

		namespace, err := resolveNamespaceFromFilesystem()
		require.NoError(t, err)
		assert.Equal(t, "kube-ns", namespace)
	})

	t.Run("no namespace source is an error", func(t *testing.T) {
		hideNamespaceSources(t)

		_, err := resolveNamespaceFromFilesystem()
		require.Error(t, err)
	})
}

func TestResolveKubernetesNamespace_EmptyConfiguredDelegatesToResolver(t *testing.T) {
	// an empty configured namespace always goes through the resolver; when
	// the resolver has no namespace source the store creation must fail
	// instead of silently picking a namespace
	hideNamespaceSources(t)

	ctx := appctx.WithContainer(t.Context())
	_, err := resolveKubernetesNamespace(ctx, log.NewLogger(), "mystore", "")
	require.Error(t, err)
}

func TestResolveKubernetesNamespace_CachedPerContext(t *testing.T) {
	client := fake.NewSimpleClientset()
	overrideKubernetesClient(t, client)

	calls := 0
	original := namespaceResolver
	namespaceResolver = func() (string, error) {
		calls++

		return "cached-ns", nil
	}
	t.Cleanup(func() { namespaceResolver = original })

	config := cfg.New(map[string]any{
		"kvstore": map[string]any{
			"one": map[string]any{},
			"two": map[string]any{},
		},
	})

	ctx := appctx.WithContainer(t.Context())
	_, err := NewConfigMapKvStore[configMapConfigurableItem](ctx, config, log.NewLogger(), &Settings{ModelId: mdl.ModelId{Name: "one"}})
	require.NoError(t, err)
	_, err = NewConfigMapKvStore[configMapConfigurableItem](ctx, config, log.NewLogger(), &Settings{ModelId: mdl.ModelId{Name: "two"}})
	require.NoError(t, err)

	// the resolved namespace is cached in the app context: the resolver runs
	// once for both stores of the same context
	assert.Equal(t, 1, calls)

	// a different context resolves again (fresh cache)
	freshCtx := appctx.WithContainer(t.Context())
	_, err = NewConfigMapKvStore[configMapConfigurableItem](freshCtx, config, log.NewLogger(), &Settings{ModelId: mdl.ModelId{Name: "one"}})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}
