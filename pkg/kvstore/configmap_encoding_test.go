package kvstore

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSupportedEncodingFormats(t *testing.T) {
	assert.Contains(t, SupportedEncodingFormats(), ConfigMapEncodingCSV)
	assert.Contains(t, SupportedEncodingFormats(), ConfigMapEncodingJSON)
	assert.Contains(t, SupportedEncodingFormats(), ConfigMapEncodingBase64)
	assert.ElementsMatch(t, []string{"csv", "json", "base64"}, SupportedEncodingFormats())
}

func TestEncodingFormat(t *testing.T) {
	assert.Empty(t, (&EncodingSettings{}).EncodingFormat())

	var nilSettings *EncodingSettings
	assert.Empty(t, nilSettings.EncodingFormat())

	assert.Equal(t, ConfigMapEncodingBase64, (&EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingBase64,
	}).EncodingFormat())

	// a disabled setting reports no format even if a format is set
	assert.Empty(t, (&EncodingSettings{
		Enabled: false,
		Format:  ConfigMapEncodingCSV,
	}).EncodingFormat())
}

func TestValidateEncodingSettings(t *testing.T) {
	// nil and disabled settings are always valid, even with garbage in the
	// unused fields
	assert.NoError(t, validateEncodingSettings(nil))
	assert.NoError(t, validateEncodingSettings(&EncodingSettings{
		Enabled: false,
		Format:  "",
	}))
	assert.NoError(t, validateEncodingSettings(&EncodingSettings{
		Enabled: false,
		Format:  "garbage",
	}))

	// enabled with each supported format
	for _, format := range SupportedEncodingFormats() {
		assert.NoError(t, validateEncodingSettings(&EncodingSettings{
			Enabled: true,
			Format:  format,
		}), "format %s must be accepted", format)
	}

	// enabled with an unknown format is rejected with a descriptive error
	err := validateEncodingSettings(&EncodingSettings{
		Enabled: true,
		Format:  "yaml",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid encoding format "yaml"`)
	assert.Contains(t, err.Error(), "csv")
	assert.Contains(t, err.Error(), "json")
	assert.Contains(t, err.Error(), "base64")
}

func TestNewConfigMapKvStore_EncodingValidation(t *testing.T) {
	t.Parallel()

	client := fake.NewSimpleClientset()

	// unknown format is rejected before the store is usable
	_, err := NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Encoding: &EncodingSettings{
			Enabled: true,
			Format:  "hex",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid encoding format "hex"`)

	// each supported format is accepted
	for _, format := range SupportedEncodingFormats() {
		_, err = NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
			Encoding: &EncodingSettings{
				Enabled: true,
				Format:  format,
			},
		})
		assert.NoError(t, err, "format %s must be accepted at construction", format)
	}

	// disabled encoding is accepted even with an unset or garbage format
	_, err = NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Encoding: &EncodingSettings{},
	})
	assert.NoError(t, err)

	_, err = NewConfigMapKvStoreWithClient[string](client, "test-ns", "test", &ConfigMapSettings{
		Encoding: &EncodingSettings{
			Enabled: false,
			Format:  "garbage",
		},
	})
	assert.NoError(t, err)
}

func TestEncodeValue_DisabledReturnsInputUnchanged(t *testing.T) {
	data := "some value"

	out, err := encodeValue(nil, []byte(data))
	require.NoError(t, err)
	assert.Equal(t, data, out)

	out, err = encodeValue(&EncodingSettings{Enabled: false, Format: ConfigMapEncodingBase64}, []byte(data))
	require.NoError(t, err)
	assert.Equal(t, data, out)

	decoded, err := decodeValue(nil, data)
	require.NoError(t, err)
	assert.Equal(t, []byte(data), decoded)

	decoded, err = decodeValue(&EncodingSettings{Enabled: false}, data)
	require.NoError(t, err)
	assert.Equal(t, []byte(data), decoded)
}

func TestEncodeValue_InvalidFormatError(t *testing.T) {
	// an unknown format must fail loudly instead of storing an unencoded or
	// half-encoded value
	out, err := encodeValue(&EncodingSettings{
		Enabled: true,
		Format:  "nope",
	}, []byte("some value"))
	require.Error(t, err)
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), `invalid encoding format "nope"`)

	decoded, err := decodeValue(&EncodingSettings{
		Enabled: true,
		Format:  "nope",
	}, "some value")
	require.Error(t, err)
	assert.Nil(t, decoded)
	assert.Contains(t, err.Error(), `invalid encoding format "nope"`)
}

func TestEncodeValue_RoundTrip(t *testing.T) {
	// payloads of the lossless formats: base64 and csv handle arbitrary
	// bytes, json only valid utf-8, so the random binary payload is covered
	// separately by TestEncodeValue_Base64ArbitraryBytes and
	// TestEncodeValue_CSVFormat
	payloads := map[string][]byte{
		"empty":            {},
		"tiny":             []byte("a"),
		"plain":            []byte("hello world"),
		"csv hostile":      []byte("a,b,c"),
		"quote":            []byte(`say "hi"`),
		"embedded newline": []byte("line1\nline2"),
		"nuls":             []byte("a\x00b\x00c"),
		"unicode":          []byte("héllo wörld ✓"),
	}

	formats := []string{ConfigMapEncodingBase64, ConfigMapEncodingJSON, ConfigMapEncodingCSV}

	for _, format := range formats {
		for payloadName, data := range payloads {
			// the csv format does not support carriage returns; that is
			// covered by its own dedicated test
			if format == ConfigMapEncodingCSV && strings.ContainsRune(string(data), '\r') {
				continue
			}

			t.Run(fmt.Sprintf("%s/%s", format, payloadName), func(t *testing.T) {
				settings := &EncodingSettings{
					Enabled: true,
					Format:  format,
				}

				stored, err := encodeValue(settings, data)
				require.NoError(t, err)

				out, err := decodeValue(settings, stored)
				require.NoError(t, err)
				assert.Equal(t, data, out)
			})
		}
	}
}

func TestEncodeValue_Base64ArbitraryBytes(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingBase64,
	}

	// every possible byte value must survive, because base64 is the
	// lossless format required for compressed values
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i)
	}

	stored, err := encodeValue(settings, data)
	require.NoError(t, err)

	// standard base64 alphabet only: the stored value is plain ascii
	assert.True(t, strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			return r
		}

		return -1
	}, stored) == stored, "stored value %q contains non base64 characters", stored)

	// the base64 of the full 256 byte value is deterministic and
	// 4 * ceil(256 / 3) = 344 chars
	assert.Equal(t, (len(data)+2)/3*4, len(stored))

	out, err := decodeValue(settings, stored)
	require.NoError(t, err)
	assert.Equal(t, data, out)

	// a larger random binary payload must survive too
	random := randomBytes(10 * 1024)
	stored, err = encodeValue(settings, random)
	require.NoError(t, err)

	out, err = decodeValue(settings, stored)
	require.NoError(t, err)
	assert.Equal(t, random, out)
}

func TestEncodeValue_JSONFormat(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingJSON,
	}

	// the json format stores the bytes as a compact json string value
	stored, err := encodeValue(settings, []byte(`a "quoted", word`))
	require.NoError(t, err)
	assert.Equal(t, `"a \"quoted\", word"`, stored)

	// control characters are escaped, so the stored value is a single line
	stored, err = encodeValue(settings, []byte("a\nb\tc"))
	require.NoError(t, err)
	assert.NotContains(t, stored, "\n")
	assert.NotContains(t, stored, "\t")

	out, err := decodeValue(settings, stored)
	require.NoError(t, err)
	assert.Equal(t, []byte("a\nb\tc"), out)

	// invalid utf-8 is replaced with the utf-8 replacement character by the
	// json encoder, so the json format is lossy for binary: base64 is the
	// format that carries arbitrary bytes losslessly
	out, err = decodeValue(settings, func() string {
		s, e := encodeValue(settings, []byte{0x61, 0x80, 0x62})
		require.NoError(t, e)

		return s
	}())
	require.NoError(t, err)
	assert.Equal(t, []byte("a\ufffdb"), out)
}

func TestDecodeValue_JSONRejectsNonStringValues(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingJSON,
	}

	// a json-encoded entry must be a json string value; anything else is a
	// corruption or a plain value stored without encoding and must be
	// rejected, not silently misread. (null is not listed on purpose: it is
	// not a string value, but json.Unmarshal treats it as the absence of a
	// value, so the empty string is decoded.)
	for _, stored := range []string{
		`42`,
		`true`,
		`["a"]`,
		`{"a": 1}`,
		`not json at all`,
		`"a" trailing`,
	} {
		out, err := decodeValue(settings, stored)
		require.Error(t, err, "stored value %q must be rejected", stored)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "json decode")
	}
}

func TestEncodeValue_CSVFormat(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingCSV,
	}

	// plain values are stored bare, without quotes
	stored, err := encodeValue(settings, []byte("hello world"))
	require.NoError(t, err)
	assert.Equal(t, "hello world", stored)

	// values with commas or quotes are quoted, embedded quotes doubled
	stored, err = encodeValue(settings, []byte(`a,b "c"`))
	require.NoError(t, err)
	assert.Equal(t, `"a,b ""c"""`, stored)

	// embedded newlines are quoted and preserved
	stored, err = encodeValue(settings, []byte("line1\nline2"))
	require.NoError(t, err)
	assert.Equal(t, `"line1
line2"`, stored)

	// the empty value is the quoted empty field, because the csv writer
	// would emit a bare newline that the csv reader does not parse back
	stored, err = encodeValue(settings, []byte(""))
	require.NoError(t, err)
	assert.Equal(t, `""`, stored)

	out, err := decodeValue(settings, `""`)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), out)

	// a random binary payload (minus carriage returns, which the csv format
	// does not support) must round trip losslessly
	random := []byte{0xff, 0xfe, 0x00, 0x0a, 0x7f, 0x2c, 0x22}
	random = append(random, randomBytes(1024)...)
	random = []byte(strings.ReplaceAll(string(random), "\r", " "))
	require.NotContains(t, string(random), "\r")

	stored, err = encodeValue(settings, random)
	require.NoError(t, err)

	out, err = decodeValue(settings, stored)
	require.NoError(t, err)
	assert.Equal(t, random, out)
}

func TestEncodeValue_CSVDeterministic(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingCSV,
	}

	for _, data := range [][]byte{
		[]byte("plain"),
		[]byte(`a,b "c"`),
		[]byte("line1\nline2"),
		{0x00, 0xff, 0x80},
	} {
		first, err := encodeValue(settings, data)
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			again, err := encodeValue(settings, data)
			require.NoError(t, err)
			assert.Equal(t, first, again, "csv encoding must be deterministic")
		}
	}
}

func TestEncodeValue_CSVRejectsCarriageReturn(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingCSV,
	}

	// the csv reader normalizes CR LF to LF, so values containing a bare or
	// trailing CR could not be recovered losslessly and are rejected
	for _, data := range [][]byte{
		[]byte("a\rb"),
		[]byte("a\r\nb"),
		[]byte("trailing\r"),
		[]byte("\r"),
	} {
		out, err := encodeValue(settings, data)
		require.Error(t, err, "value %q must be rejected", data)
		assert.Empty(t, out)
		assert.Contains(t, err.Error(), "carriage return")
	}
}

func TestDecodeValue_CSVRejectsForeignDocuments(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingCSV,
	}

	// a csv-encoded entry is a single-row document with exactly one field;
	// multi-field rows and unparsable input must be rejected
	for _, stored := range []string{
		"a,b",
		`"a",b`,
		"a\nb",
		`"unterminated`,
	} {
		out, err := decodeValue(settings, stored)
		require.Error(t, err, "stored value %q must be rejected", stored)
		assert.Nil(t, out)
		assert.Contains(t, err.Error(), "csv decode")
	}

	// an empty entry parses to no record at all
	out, err := decodeValue(settings, "")
	require.Error(t, err)
	assert.Nil(t, out)
}

func TestEncodeValue_Base64RejectsCorruptInput(t *testing.T) {
	settings := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingBase64,
	}

	// not valid base64 at all
	out, err := decodeValue(settings, "not valid base64!!!")
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "base64")

	// right length but bad padding
	out, err = decodeValue(settings, "ab")
	require.Error(t, err)
	assert.Nil(t, out)
}

// TestEncodeValue_AfterCompression exercises the documented composition of
// both layers: compression is applied first, encoding on top of the
// compressed bytes, and the order is reversed on the way back in. This is
// the helper-level proof that the two layers compose; the store-level wiring
// is covered by the integration tests.
func TestEncodeValue_AfterCompression(t *testing.T) {
	compression := &CompressionSettings{
		Enabled: true,
		Algo:    ConfigMapCompressionGzip,
		Level:   gzip.DefaultCompression,
	}
	encoding := &EncodingSettings{
		Enabled: true,
		Format:  ConfigMapEncodingBase64,
	}

	for _, data := range [][]byte{
		{},
		[]byte("a"),
		bytes.Repeat([]byte("compressible payload "), 64),
		randomBytes(10 * 1024),
	} {
		// write path: marshal -> compress -> encode
		compressed, err := compressValue(compression, data)
		require.NoError(t, err)

		stored, err := encodeValue(encoding, compressed)
		require.NoError(t, err)

		// the encoded value must be a plain string that the compressed
		// binary would never be: no NUL bytes, no high bytes
		assert.True(t, strings.Map(func(r rune) rune {
			if r >= 0x20 && r < 0x7f {
				return r
			}

			return -1
		}, stored) == stored, "stored value %q is not printable ascii", stored)

		// read path: decode -> decompress
		decoded, err := decodeValue(encoding, stored)
		require.NoError(t, err)
		assert.Equal(t, compressed, decoded)

		out, err := decompressValue(compression, decoded)
		require.NoError(t, err)
		assert.Equal(t, data, out)
	}
}
