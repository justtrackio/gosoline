package kvstore

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSupportedCompressionAlgos(t *testing.T) {
	assert.Contains(t, SupportedCompressionAlgos(), ConfigMapCompressionGzip)
	assert.Contains(t, SupportedCompressionAlgos(), "gzip")
}

func TestValidCompressionLevels(t *testing.T) {
	assert.Equal(t, []int{-1, 0, 1, 9}, ValidCompressionLevels(ConfigMapCompressionGzip))
	assert.Empty(t, ValidCompressionLevels("unknown"))
}

func TestCompressionAlgo(t *testing.T) {
	assert.Empty(t, (&CompressionSettings{}).CompressionAlgo())

	var nilSettings *CompressionSettings
	assert.Empty(t, nilSettings.CompressionAlgo())

	assert.Equal(t, ConfigMapCompressionGzip, (&CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
	}).CompressionAlgo())
}

func TestValidateCompressionSettings(t *testing.T) {
	// nil and disabled settings are always valid, even with garbage in the
	// unused fields
	assert.NoError(t, validateCompressionSettings(nil))
	assert.NoError(t, validateCompressionSettings(&CompressionSettings{
		Enabled: false,
		Algo:    "",
		Level:   42,
	}))

	// enabled with a supported algorithm and valid levels
	assert.NoError(t, validateCompressionSettings(&CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
		Level:   gzip.DefaultCompression,
	}))
	for _, level := range ValidCompressionLevels(ConfigMapCompressionGzip) {
		assert.NoError(t, validateCompressionSettings(&CompressionSettings{
			Enabled: true,
			Algo:    ConfigMapCompressionGzip,
			Level:   level,
		}))
	}

	// enabled with an unknown algorithm
	err := validateCompressionSettings(&CompressionSettings{
		Enabled: true,
		Algo:    "lz4",
		Level:   gzip.DefaultCompression,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid compression algorithm "lz4"`)
	assert.Contains(t, err.Error(), "gzip")

	// enabled with a valid algorithm but invalid level
	for _, level := range []int{-2, 2, 5, 10, 255} {
		err := validateCompressionSettings(&CompressionSettings{
			Enabled: true,
			Algo:    ConfigMapCompressionGzip,
			Level:   level,
		})
		require.Error(t, err, "level %d must be rejected", level)
		assert.Contains(t, err.Error(), fmt.Sprintf("invalid compression level %d", level))
		assert.Contains(t, err.Error(), "gzip")
	}
}

func TestNewConfigMapKvStore_CompressionValidation(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()

	// invalid algorithm is rejected before the store is usable
	_, err := NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Compression: &CompressionSettings{
			Enabled: true,
			Algo:    "zstd",
			Level:   gzip.DefaultCompression,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid compression algorithm")

	// invalid level is rejected before the store is usable
	_, err = NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Compression: &CompressionSettings{
			Enabled: true,
			Algo:    ConfigMapCompressionGzip,
			Level:   7,
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid compression level")

	// disabled compression is accepted even with unset fields
	_, err = NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Compression: &CompressionSettings{},
	})
	assert.NoError(t, err)
}

func TestCompressValue_DisabledReturnsInputUnchanged(t *testing.T) {
	data := []byte("some value")

	out, err := compressValue(nil, data)
	require.NoError(t, err)
	assertSameSlice(t, data, out)

	out, err = compressValue(&CompressionSettings{Enabled: false, Algo: ConfigMapCompressionGzip, Level: 9}, data)
	require.NoError(t, err)
	assertSameSlice(t, data, out)

	out, err = decompressValue(nil, data)
	require.NoError(t, err)
	assertSameSlice(t, data, out)

	out, err = decompressValue(&CompressionSettings{Enabled: false}, data)
	require.NoError(t, err)
	assertSameSlice(t, data, out)
}

// assertSameSlice asserts that a and b are equal and share the same backing
// array, i.e. the helper returned the input slice untouched.
func assertSameSlice(t *testing.T, input, out []byte) {
	t.Helper()

	require.Equal(t, input, out)

	if len(input) > 0 {
		assert.Truef(t, &out[0] == &input[0], "expected the input slice to be returned unchanged")
	}
}

func TestCompressValue_InvalidSettingsError(t *testing.T) {
	data := []byte("some value")

	// compression is a lossy-looking operation from the caller's perspective,
	// so it must fail loudly instead of writing raw or half-compressed bytes
	out, err := compressValue(&CompressionSettings{
		Enabled: true,
		Algo:    "nope",
		Level:   gzip.DefaultCompression,
	}, data)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "invalid compression algorithm")

	out, err = decompressValue(&CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
		Level:   3,
	}, data)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "invalid compression level")
}

func TestCompressValue_RoundTrip(t *testing.T) {
	levelBoundaries := map[string]int{
		"default (-1)":         gzip.DefaultCompression,
		"no compression (0)":   gzip.NoCompression,
		"best speed (1)":       gzip.BestSpeed,
		"best compression (9)": gzip.BestCompression,
	}

	// compressible and incompressible payloads, plus the empty edge case
	payloads := map[string][]byte{
		"empty":         {},
		"tiny":          []byte("a"),
		"compressible":  bytes.Repeat([]byte("gosoline"), 64),
		"random 10 KiB": randomBytes(10 * 1024),
	}

	for name, level := range levelBoundaries {
		for payloadName, data := range payloads {
			t.Run(fmt.Sprintf("%s/%s", name, payloadName), func(t *testing.T) {
				settings := &CompressionSettings{
					Enabled: true,
					Algo:    ConfigMapCompressionGzip,
					Level:   level,
				}

				compressed, err := compressValue(settings, data)
				require.NoError(t, err)
				require.NotEmpty(t, compressed)

				// the compressed output must not be the raw payload, because
				// the gzip frame header changes even incompressible data
				assert.NotEqual(t, data, compressed)

				// a plain base64 string round trip must survive the encoding
				// layer boundary: the compressed bytes become a string and
				// back before decompression
				decoded := string(compressed)
				require.NotEmpty(t, decoded)

				out, err := decompressValue(settings, []byte(decoded))
				require.NoError(t, err)
				assert.Equal(t, data, out)
			})
		}
	}
}

func TestCompressValue_ActuallyCompresses(t *testing.T) {
	data := bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog "), 128)

	out, err := compressValue(&CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
		Level:   gzip.BestCompression,
	}, data)
	require.NoError(t, err)
	assert.Less(t, len(out), len(data))
}

func TestDecompressValue_CorruptData(t *testing.T) {
	settings := &CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
		Level:   gzip.DefaultCompression,
	}

	// plain text is not a gzip stream
	_, err := decompressValue(settings, []byte("definitely not gzip"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gzip")

	// a truncated gzip stream is rejected, not silently truncated
	valid, err := compressValue(settings, bytes.Repeat([]byte("x"), 1024))
	require.NoError(t, err)
	_, err = decompressValue(settings, valid[:len(valid)/2])
	require.Error(t, err)
}

func randomBytes(n int) []byte {
	data := make([]byte, n)
	_, _ = rand.New(rand.NewSource(42)).Read(data)

	return data
}
