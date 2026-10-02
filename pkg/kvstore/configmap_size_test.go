package kvstore_test

import (
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sizeTablePayload is a 1kb value: an Item whose marshaled JSON form is
// exactly 1024 bytes long (25 bytes of json skeleton, including the 6 byte
// id, plus a 999 byte body).
func sizeTablePayload() Item {
	return Item{Id: "sample", Body: strings.Repeat("a", 999)}
}

// TestConfigMapKvStore_SizeComparisonTable verifies the stored-size numbers
// published in the godoc size comparison table of the ConfigMap backed
// KvStore. For each valid configuration the table documents, it puts exactly
// 1kb of data through the real store and asserts the size of the value that
// ends up in the backing ConfigMap, so the documented table cannot drift from
// the actual behavior.
//
// The payload is a highly compressible document (a long run of identical
// bytes), which is representative for the store's intended use-cases (e.g.
// metadata refreshes) and makes the compression savings visible.
func TestConfigMapKvStore_SizeComparisonTable(t *testing.T) {
	// the sizes below are the values published in the doc comment of
	// configmap.go; keep the two in sync.
	type sizeCase struct {
		name     string
		settings *kvstore.ConfigMapSettings
		stored   int
	}

	cases := []sizeCase{
		{name: "plain", settings: &kvstore.ConfigMapSettings{}, stored: 1024},
		{name: "base64 encoding only", settings: &kvstore.ConfigMapSettings{
			Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
		}, stored: 1368},
		{name: "json encoding only", settings: &kvstore.ConfigMapSettings{
			Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingJSON},
		}, stored: 1034},
		{name: "csv encoding only", settings: &kvstore.ConfigMapSettings{
			Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingCSV},
		}, stored: 1034},
		{name: "gzip level 1 + base64", settings: &kvstore.ConfigMapSettings{
			Compression: &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 1},
			Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
		}, stored: 88},
		{name: "gzip level 9 + base64", settings: &kvstore.ConfigMapSettings{
			Compression: &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 9},
			Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
		}, stored: 76},
	}

	value := sizeTablePayload()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, store, client := buildIntegrationConfigMapStore[Item](t, tc.settings, nil)

			require.NoError(t, store.Put(ctx, "size", value))

			stored := getKeyStoredValue(t, client, ctx, "size")
			assert.Len(t, stored, tc.stored, "documented stored size %d must match the actually stored size", tc.stored)

			// the value must survive the round trip under this configuration
			item := &Item{}
			found, err := store.Get(ctx, "size", item)
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, value, *item)
		})
	}
}

// TestConfigMapKvStore_SizeComparisonTable_Ordering asserts the relative
// ordering of the size comparison table, which is what matters when choosing
// a configuration: compressing the 1kb payload with base64 on top shrinks it
// to well under 10% of the plain size, while the encoding-only options never
// shrink the value (base64 grows it, json and csv only add a small framing
// overhead).
func TestConfigMapKvStore_SizeComparisonTable_Ordering(t *testing.T) {
	clientSizes := map[string]int{}
	stores := map[string]*kvstore.ConfigMapSettings{
		"plain":  {},
		"base64": {Encoding: &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64}},
		"gzip": {
			Compression: &kvstore.CompressionSettings{Enabled: true, Algo: kvstore.ConfigMapCompressionGzip, Level: 9},
			Encoding:    &kvstore.EncodingSettings{Enabled: true, Format: kvstore.ConfigMapEncodingBase64},
		},
	}

	value := sizeTablePayload()
	for name, settings := range stores {
		ctx, store, client := buildIntegrationConfigMapStore[Item](t, settings, nil)
		require.NoError(t, store.Put(ctx, "size", value))
		clientSizes[name] = len(getKeyStoredValue(t, client, ctx, "size"))
	}

	// the compressible payload shrinks dramatically with gzip + base64
	assert.Less(t, clientSizes["gzip"], clientSizes["plain"]/10)
	// base64 without compression grows the value
	assert.Greater(t, clientSizes["base64"], clientSizes["plain"])
}
