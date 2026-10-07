package kvstore_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	configMapTestNamespace = "test-ns"
	configMapTestStore     = "test"
	// "kvstore-test-" + 54 chars = 63 chars, the kubernetes spec name length
	// limit for the per-key configmap name
	configMapMaxTestKeyLength = kvstore.ConfigMapMaxKeyLength - len("kvstore-test-")
)

// buildTestableConfigMapStore creates a ConfigMap backed KvStore on a fresh
// fake client. A non-nil data map pre-seeds a key's dedicated ConfigMap: the
// data map is used verbatim as that ConfigMap's data, and the ConfigMap is
// named "kvstore-<store>-<first data key>".
func buildTestableConfigMapStore[T any](t *testing.T, data map[string]string) (context.Context, kvstore.KvStore[T], *fake.Clientset) {
	t.Helper()

	client := fake.NewSimpleClientset()

	for key, value := range data {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("kvstore-%s-%s", configMapTestStore, key),
				Namespace: configMapTestNamespace,
			},
			Data: map[string]string{key: value},
		}
		_, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Create(t.Context(), cm, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	store, err := kvstore.NewConfigMapKvStoreWithClient[T](client, configMapTestNamespace, configMapTestStore, &kvstore.ConfigMapSettings{
		BatchSize: 100,
	})
	require.NoError(t, err)

	return t.Context(), store, client
}

// getKeyConfigMap reads the dedicated ConfigMap of the given key from the
// fake client.
func getKeyConfigMap(t *testing.T, client *fake.Clientset, ctx context.Context, key string) *corev1.ConfigMap {
	t.Helper()

	cm, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, fmt.Sprintf("kvstore-%s-%s", configMapTestStore, key), metav1.GetOptions{})
	require.NoError(t, err)

	return cm
}

func TestConfigMapKvStore_PutAndGet(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	err := store.Put(ctx, "foo", Item{Id: "foo", Body: "bar"})
	require.NoError(t, err)

	// each key gets its own dedicated configmap, named
	// kvstore-<storeName>-<keyName>, holding the value under the key name
	cm := getKeyConfigMap(t, client, ctx, "foo")
	assert.Equal(t, configMapTestNamespace, cm.Namespace)
	require.Contains(t, cm.Data, "foo")
	assert.Equal(t, `{"id":"foo","body":"bar"}`, cm.Data["foo"])

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "foo", item.Id)
	assert.Equal(t, "bar", item.Body)
}

func TestConfigMapKvStore_GetMissing(t *testing.T) {
	// the key has no configmap: it is reported as missing, not an error
	ctx, store, _ := buildTestableConfigMapStore[Item](t, nil)

	item := &Item{}
	found, err := store.Get(ctx, "missing", item)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, Item{}, *item)
}

func TestConfigMapKvStore_Contains(t *testing.T) {
	ctx, store, _ := buildTestableConfigMapStore[Item](t, map[string]string{
		"bar": `{"id":"bar","body":"baz"}`,
	})

	exists, err := store.Contains(ctx, "bar")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Contains(ctx, "foo")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestConfigMapKvStore_Delete(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, map[string]string{
		"foo": `{"id":"foo","body":"bar"}`,
	})

	err := store.Delete(ctx, "foo")
	require.NoError(t, err)

	// deleting a key removes its dedicated configmap
	_, err = client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-foo", metav1.GetOptions{})
	require.Error(t, err)

	found, err := store.Contains(ctx, "foo")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestConfigMapKvStore_DeleteMissingConfigMap(t *testing.T) {
	ctx, store, _ := buildTestableConfigMapStore[Item](t, nil)

	// deleting a key whose configmap does not exist yet is a no-op
	err := store.Delete(ctx, "foo")
	require.NoError(t, err)
}

func TestConfigMapKvStore_PutBatch(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	err := store.PutBatch(ctx, map[string]Item{
		"foo": {Id: "foo", Body: "bar"},
		"fuu": {Id: "fuu", Body: "baz"},
	})
	require.NoError(t, err)

	// each key of the batch is stored in its own dedicated configmap
	assert.Equal(t, `{"id":"foo","body":"bar"}`, getKeyConfigMap(t, client, ctx, "foo").Data["foo"])
	assert.Equal(t, `{"id":"fuu","body":"baz"}`, getKeyConfigMap(t, client, ctx, "fuu").Data["fuu"])

	result := map[string]Item{}
	missing, err := store.GetBatch(ctx, []string{"foo", "fuu"}, result)
	require.NoError(t, err)
	assert.Empty(t, missing)
	assert.Equal(t, "bar", result["foo"].Body)
	assert.Equal(t, "baz", result["fuu"].Body)
}

func TestConfigMapKvStore_GetBatch(t *testing.T) {
	ctx, store, _ := buildTestableConfigMapStore[Item](t, map[string]string{
		"foo": `{"id":"foo","body":"bar"}`,
	})

	result := map[string]Item{}
	missing, err := store.GetBatch(ctx, []string{"foo", "fuu"}, result)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	assert.Contains(t, missing, "fuu")
	assert.Equal(t, "bar", result["foo"].Body)
	assert.NotContains(t, result, "fuu")
}

func TestConfigMapKvStore_GetBatchInvalidKey(t *testing.T) {
	// the key exceeds the configmap name length limit
	ctx, store, _ := buildTestableConfigMapStore[Item](t, nil)

	longKey := strings.Repeat("a", configMapMaxTestKeyLength+1)

	_, err := store.GetBatch(ctx, []string{longKey}, map[string]Item{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")
}

func TestConfigMapKvStore_DeleteBatch(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, map[string]string{
		"foo": `{"id":"foo","body":"bar"}`,
		"fuu": `{"id":"fuu","body":"baz"}`,
	})

	// a key with no configmap may be mixed in and is a no-op
	err := store.DeleteBatch(ctx, []string{"foo", "fuu", "missing"})
	require.NoError(t, err)

	for _, key := range []string{"foo", "fuu"} {
		_, gerr := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, fmt.Sprintf("kvstore-%s-%s", configMapTestStore, key), metav1.GetOptions{})
		require.Error(t, gerr, "configmap for key %s must be deleted", key)
	}
}

func TestConfigMapKvStore_EstimateSize(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	// two keys of this store
	require.NoError(t, store.Put(ctx, "foo", Item{Id: "foo", Body: "bar"}))
	require.NoError(t, store.Put(ctx, "fuu", Item{Id: "fuu", Body: "baz"}))

	// configmaps that are not part of this store must not be counted
	_, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "kvstore-other-bar", Namespace: configMapTestNamespace},
		Data:       map[string]string{"bar": "x"},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = client.CoreV1().ConfigMaps(configMapTestNamespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: configMapTestNamespace},
		Data:       map[string]string{"value": "v"},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	sizedStore, ok := store.(kvstore.SizedStore[Item])
	require.True(t, ok)

	assert.Equal(t, int64(2), *sizedStore.EstimateSize())
}

func TestConfigMapKvStore_KeyName(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	err := store.Put(ctx, "some-key", Item{Id: "some-key", Body: "body"})
	require.NoError(t, err)

	// the key is stored in a dedicated configmap following the pattern
	// kvstore-<storeName>-<keyName>
	cm, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-some-key", metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, cm.Data, 1)
	assert.Contains(t, cm.Data, "some-key")
}

func TestConfigMapKvStore_KeyLengthValidation(t *testing.T) {
	ctx, store, _ := buildTestableConfigMapStore[Item](t, nil)

	// the maximum length fits exactly
	maxKey := strings.Repeat("a", configMapMaxTestKeyLength)
	err := store.Put(ctx, maxKey, Item{Id: maxKey, Body: "bar"})
	require.NoError(t, err)

	found, err := store.Contains(ctx, maxKey)
	require.NoError(t, err)
	assert.True(t, found)

	// one character too long is rejected
	tooLong := strings.Repeat("a", configMapMaxTestKeyLength+1)

	_, err = store.Get(ctx, tooLong, &Item{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")

	err = store.Put(ctx, tooLong, Item{Id: tooLong, Body: "bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")

	err = store.Delete(ctx, tooLong)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")

	exists, err := store.Contains(ctx, tooLong)
	require.Error(t, err)
	assert.False(t, exists)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")
}

func TestConfigMapKvStore_EmptyKey(t *testing.T) {
	ctx, store, _ := buildTestableConfigMapStore[Item](t, nil)

	err := store.Put(ctx, "", Item{Id: "", Body: "bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")

	exists, err := store.Contains(ctx, "")
	require.Error(t, err)
	assert.False(t, exists)
}

func TestConfigMapKvStore_ConstructorValidation(t *testing.T) {
	client := fake.NewSimpleClientset()

	_, err := kvstore.NewConfigMapKvStoreWithClient[Item](nil, configMapTestNamespace, configMapTestStore, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is required")

	_, err = kvstore.NewConfigMapKvStoreWithClient[Item](client, "", configMapTestStore, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace must not be empty")

	_, err = kvstore.NewConfigMapKvStoreWithClient[Item](client, strings.Repeat("n", 64), configMapTestStore, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the kubernetes spec name length limit")

	// the configmap name starts with "kvstore-<storeName>-", so the store
	// name must leave at least one character for the key: 63 - 7 - 2 = 54
	// chars
	_, err = kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, strings.Repeat("s", 56), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum store name length")

	// nil settings are accepted and defaulted
	store, err := kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, configMapTestStore, nil)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestConfigMapKvStore_MissingConfigMapIsCreated(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	// no configmap for the key exists yet
	_, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-foo", metav1.GetOptions{})
	require.Error(t, err)

	err = store.Put(ctx, "foo", Item{Id: "foo", Body: "bar"})
	require.NoError(t, err)

	cm := getKeyConfigMap(t, client, ctx, "foo")
	assert.Equal(t, configMapTestNamespace, cm.Namespace)
	assert.Equal(t, map[string]string{
		"foo": `{"id":"foo","body":"bar"}`,
	}, cm.Data)
}

func TestConfigMapKvStore_DataSizeLimit(t *testing.T) {
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	// a single value that pushes its own configmap's data map beyond the
	// 1MiB limit is rejected. The limit now applies to the single key, not
	// to the whole store.
	hugeBody := strings.Repeat("x", kvstore.ConfigMapMaxDataSize)
	err := store.Put(ctx, "huge", Item{Id: "huge", Body: hugeBody})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the kubernetes configmap size limit")

	// the key's configmap was not created
	_, gerr := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-huge", metav1.GetOptions{})
	require.Error(t, gerr)
}

func TestConfigMapKvStore_NoNoisyNeighbor(t *testing.T) {
	// one key already close to the 1MiB limit must not prevent writing
	// another key: with a shared configmap the large value would consume the
	// whole object and push the second write past the limit (noisy neighbor)
	ctx, store, client := buildTestableConfigMapStore[Item](t, nil)

	largeBody := strings.Repeat("x", 900*1024)
	require.NoError(t, store.Put(ctx, "large", Item{Id: "large", Body: largeBody}))

	// a second, small key must still fit in its own configmap even though the
	// first key's configmap is near the size limit
	require.NoError(t, store.Put(ctx, "small", Item{Id: "small", Body: "ok"}))

	assert.Equal(t, "ok", assertSmallBody(t, store, ctx))
	// the two keys live in separate configmaps
	_, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-large", metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().ConfigMaps(configMapTestNamespace).Get(ctx, "kvstore-test-small", metav1.GetOptions{})
	require.NoError(t, err)
}

func assertSmallBody(t *testing.T, store kvstore.KvStore[Item], ctx context.Context) string {
	t.Helper()

	item := &Item{}
	found, err := store.Get(ctx, "small", item)
	require.NoError(t, err)
	require.True(t, found)

	return item.Body
}

func TestConfigMapKvStore_UpdateError(t *testing.T) {
	client := fake.NewSimpleClientset()
	// pre-seed the key's dedicated configmap so the write goes through the
	// update path
	_, err := client.CoreV1().ConfigMaps(configMapTestNamespace).Create(t.Context(), &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "kvstore-test-foo", Namespace: configMapTestNamespace},
		Data:       map[string]string{"foo": `{"id":"foo","body":"bar"}`},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	// fail all updates to simulate concurrent modification / api errors
	client.PrependReactor("update", "configmaps", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("api error")
	})

	store, err := kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, configMapTestStore, &kvstore.ConfigMapSettings{
		BatchSize: 100,
	})
	require.NoError(t, err)

	err = store.Put(t.Context(), "foo", Item{Id: "foo", Body: "baz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can not update configmap")
}
