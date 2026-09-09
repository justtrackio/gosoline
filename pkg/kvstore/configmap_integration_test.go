package kvstore_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// buildIntegrationConfigMapStore creates a ConfigMap backed KvStore on a fake
// kubernetes client. A non-nil data map pre-seeds dedicated ConfigMaps, one
// per key: each entry's key becomes both the data key and part of the
// ConfigMap name "kvstore-<store>-<key>".
func buildIntegrationConfigMapStore[T any](t *testing.T, settings *kvstore.ConfigMapSettings, data map[string]string) (context.Context, kvstore.KvStore[T], *fake.Clientset) {
	t.Helper()

	return buildIntegrationConfigMapStoreWithClient[T](t, settings, data, nil)
}

// buildIntegrationConfigMapStoreWithClient behaves like
// buildIntegrationConfigMapStore but reuses the given fake client, so
// multiple store instances can share the same (fake) ConfigMaps.
func buildIntegrationConfigMapStoreWithClient[T any](t *testing.T, settings *kvstore.ConfigMapSettings, data map[string]string, client *fake.Clientset) (context.Context, kvstore.KvStore[T], *fake.Clientset) {
	t.Helper()

	if settings == nil {
		settings = &kvstore.ConfigMapSettings{}
	}

	settings.BatchSize = 100

	if client == nil {
		client = fake.NewSimpleClientset()
	}

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

	store, err := kvstore.NewConfigMapKvStoreWithClient[T](client, configMapTestNamespace, configMapTestStore, settings)
	require.NoError(t, err)

	return t.Context(), store, client
}

// getKeyStoredValue reads the raw stored value of a key from its dedicated
// ConfigMap.
func getKeyStoredValue(t *testing.T, client *fake.Clientset, ctx context.Context, key string) string {
	t.Helper()

	return getKeyConfigMap(t, client, ctx, key).Data[key]
}

// gzipLevel1 compresses data with gzip level 1; helper to build the expected
// raw compressed bytes for comparisons.
func gzipLevel1(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, 1)
	require.NoError(t, err)
	_, err = w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return buf.Bytes()
}

// marshalItem marshals the way the store does, so tests can build the
// expected plain representation for comparisons.
func marshalItem(t *testing.T, item Item) []byte {
	t.Helper()

	out, err := json.Marshal(item)
	require.NoError(t, err)

	return out
}

func TestConfigMapKvStore_Integration_PlainValues(t *testing.T) {
	// both compression and encoding disabled: stored data must be the raw
	// marshaled value, exactly as before the layers existed
	ctx, store, client := buildIntegrationConfigMapStore[Item](t, &kvstore.ConfigMapSettings{}, nil)

	require.NoError(t, store.Put(ctx, "foo", Item{Id: "foo", Body: "bar"}))

	stored := getKeyStoredValue(t, client, ctx, "foo")
	assert.Equal(t, string(marshalItem(t, Item{Id: "foo", Body: "bar"})), stored)

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, Item{Id: "foo", Body: "bar"}, *item)

	// values written before compression/encoding existed must still be
	// readable by a plain store
	_, store2, _ := buildIntegrationConfigMapStore[Item](t, &kvstore.ConfigMapSettings{}, map[string]string{
		"legacy": string(marshalItem(t, Item{Id: "legacy", Body: "plain json"})),
	})

	found, err = store2.Get(ctx, "legacy", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, Item{Id: "legacy", Body: "plain json"}, *item)
}

func TestConfigMapKvStore_Integration_CompressedBase64(t *testing.T) {
	settings := &kvstore.ConfigMapSettings{
		Compression: &kvstore.CompressionSettings{
			Enabled: true,
			Algo:    kvstore.ConfigMapCompressionGzip,
			Level:   1,
		},
		Encoding: &kvstore.EncodingSettings{
			Enabled: true,
			Format:  kvstore.ConfigMapEncodingBase64,
		},
	}

	ctx, store, client := buildIntegrationConfigMapStore[Item](t, settings, nil)

	value := Item{Id: "foo", Body: strings.Repeat("compressible body ", 250)}
	require.NoError(t, store.Put(ctx, "foo", value))

	stored := getKeyStoredValue(t, client, ctx, "foo")
	plain := marshalItem(t, value)

	// the stored value is exactly base64(gzip(marshaled value)), not the
	// plain json and not the raw gzip bytes
	compressed := gzipLevel1(t, plain)
	assert.Less(t, len(compressed), len(plain))
	assert.Equal(t, base64.StdEncoding.EncodeToString(compressed), stored)

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, *item)
}

func TestConfigMapKvStore_Integration_JsonEncoding(t *testing.T) {
	settings := &kvstore.ConfigMapSettings{
		Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingJSON},
	}

	ctx, store, client := buildIntegrationConfigMapStore[Item](t, settings, nil)

	value := Item{Id: "foo", Body: `he said "hi"`}
	require.NoError(t, store.Put(ctx, "foo", value))

	stored := getKeyStoredValue(t, client, ctx, "foo")
	plain := marshalItem(t, value)

	// the stored value is a json string value wrapping the plain marshaled
	// value: unmarshaling it must yield exactly the plain bytes
	assert.NotEqual(t, string(plain), stored)
	assert.True(t, strings.HasPrefix(stored, `"`))
	assert.True(t, strings.HasSuffix(stored, `"`))

	var unquoted string
	require.NoError(t, json.Unmarshal([]byte(stored), &unquoted))
	assert.Equal(t, string(plain), unquoted)

	item := &Item{}
	found, err := store.Get(ctx, "foo", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, *item)

	// a json-encoded value seeded directly into the configmap is readable by
	// a store with the same settings
	seeded := string(plain)
	seededJSON, err := json.Marshal(seeded)
	require.NoError(t, err)

	_, store2, _ := buildIntegrationConfigMapStore[Item](t, settings, map[string]string{
		"foo": string(seededJSON),
	})

	found, err = store2.Get(ctx, "foo", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, *item)
}

func TestConfigMapKvStore_Integration_CsvEncoding(t *testing.T) {
	settings := &kvstore.ConfigMapSettings{
		Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingCSV},
	}

	ctx, store, client := buildIntegrationConfigMapStore[Item](t, settings, nil)

	// value whose marshaled form contains commas and quotes: forces csv
	// quoting with doubled quotes
	value := Item{Id: "csv", Body: `He said "hi", ok`}
	require.NoError(t, store.Put(ctx, "csv", value))

	stored := getKeyStoredValue(t, client, ctx, "csv")
	plain := marshalItem(t, value)

	// the stored value is not the plain value and parses as a single-row
	// single-field csv document whose field is exactly the plain value
	assert.NotEqual(t, string(plain), stored)

	r := csv.NewReader(strings.NewReader(stored))
	record, err := r.Read()
	require.NoError(t, err)
	require.Len(t, record, 1)
	assert.Equal(t, string(plain), record[0])

	_, err = r.Read()
	require.ErrorIs(t, err, io.EOF)

	item := &Item{}
	found, err := store.Get(ctx, "csv", item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, value, *item)

	// a foreign multi-field entry in the key's configmap is rejected on read,
	// not silently misread
	_, store2, _ := buildIntegrationConfigMapStore[Item](t, settings, map[string]string{
		"foreign": `a,b`,
	})

	_, err = store2.Get(ctx, "foreign", &Item{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "single field")
}

func TestConfigMapKvStore_Integration_RoundTrip(t *testing.T) {
	cases := map[string]*kvstore.ConfigMapSettings{
		"plain": {},
		"compression+base64": {
			Compression: &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 9},
			Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
		},
		"json encoding": {Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingJSON}},
		"csv encoding":  {Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingCSV}},
		// disabled sub-settings behave like nil settings
		"disabled sub settings": {
			Compression: &kvstore.CompressionSettings{Enabled: false, Algo: kvstore.ConfigMapCompressionGzip, Level: 1},
			Encoding:    &kvstore.EncodingSettings{Enabled: false, Format: kvstore.ConfigMapEncodingBase64},
		},
	}

	for name, settings := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, store, client := buildIntegrationConfigMapStore[Item](t, settings, nil)

			values := map[string]Item{
				"unicode":     {Id: "unicode", Body: "héllo wörld 🚀"},
				"punctuation": {Id: "punctuation", Body: `He said "hi", there & left`},
				"newlines":    {Id: "newlines", Body: "line one\nline two\n"},
				"empty":       {Id: "empty", Body: ""},
				"long":        {Id: "long", Body: strings.Repeat("x", 4096)},
			}

			require.NoError(t, store.PutBatch(ctx, values))

			result := map[string]Item{}
			missing, err := store.GetBatch(ctx, []string{"unicode", "punctuation", "newlines", "empty", "long"}, result)
			require.NoError(t, err)
			assert.Empty(t, missing)
			assert.Equal(t, values, result)

			// single-key reads return the same values
			for key, value := range values {
				item := &Item{}
				found, err := store.Get(ctx, key, item)
				require.NoError(t, err)
				assert.True(t, found, "key %s", key)
				assert.Equal(t, value, *item, "key %s", key)
			}

			// a second store instance with the same settings and a shared
			// (fake) client reads the same data back from the same ConfigMaps
			_, store2, _ := buildIntegrationConfigMapStoreWithClient[Item](t, settings, nil, client)

			item := &Item{}
			found, err := store2.Get(ctx, "unicode", item)
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, values["unicode"], *item)
		})
	}
}

func TestConfigMapKvStore_Integration_InvalidCombinations(t *testing.T) {
	client := fake.NewSimpleClientset()

	compressed := &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 1}

	cases := map[string]struct {
		encoding *kvstore.EncodingSettings
		errPart  string
	}{
		"compression with nil encoding": {
			encoding: nil,
			errPart:  "compression is enabled but encoding is disabled",
		},
		"compression with disabled encoding": {
			encoding: &kvstore.EncodingSettings{Enabled: false, Format: kvstore.ConfigMapEncodingBase64},
			errPart:  "compression is enabled but encoding is disabled",
		},
		"compression with csv": {
			encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingCSV},
			errPart:  "compression requires the \"base64\" encoding format",
		},
		"compression with json": {
			encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingJSON},
			errPart:  "compression requires the \"base64\" encoding format",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, configMapTestStore, &kvstore.ConfigMapSettings{
				Compression: compressed,
				Encoding:    tc.encoding,
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errPart)
		})
	}

	// the valid combination is accepted
	valid, err := kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, configMapTestStore, &kvstore.ConfigMapSettings{
		Compression: compressed,
		Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
	})
	require.NoError(t, err)
	require.NotNil(t, valid)

	// every encoding-only combination is accepted
	for _, format := range kvstore.SupportedEncodingFormats() {
		encodingOnly, err := kvstore.NewConfigMapKvStoreWithClient[Item](client, configMapTestNamespace, configMapTestStore, &kvstore.ConfigMapSettings{
			Encoding: &kvstore.EncodingSettings{Enabled: true, Format: format},
		})
		require.NoError(t, err, "format %s", format)
		require.NotNil(t, encodingOnly)
	}
}

func TestConfigMapKvStore_Integration_KeyLengthWithCompression(t *testing.T) {
	settings := &kvstore.ConfigMapSettings{
		Compression: &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 1},
		Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
	}

	ctx, store, _ := buildIntegrationConfigMapStore[Item](t, settings, nil)

	// the maximum key length fits and round-trips even with compression and
	// encoding enabled: the limits apply to the key, not the value
	maxKey := strings.Repeat("a", configMapMaxTestKeyLength)
	require.NoError(t, store.Put(ctx, maxKey, Item{Id: maxKey, Body: "bar"}))

	item := &Item{}
	found, err := store.Get(ctx, maxKey, item)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "bar", item.Body)

	tooLong := strings.Repeat("a", configMapMaxTestKeyLength+1)

	_, err = store.Get(ctx, tooLong, &Item{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")

	err = store.Put(ctx, tooLong, Item{Id: tooLong, Body: "bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")

	_, err = store.GetBatch(ctx, []string{tooLong}, map[string]Item{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the maximum key name length")
}
