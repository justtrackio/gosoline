package base64_test

import (
	"github.com/applike/gosoline/pkg/encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testVector struct {
	plain   string
	encoded string
}

var testVectors = []testVector{
	{
		plain:   "",
		encoded: "",
	},
	{
		plain:   "a",
		encoded: "YQ==",
	},
	{
		plain:   "Hi",
		encoded: "SGk=",
	},
	{
		plain:   "Foo",
		encoded: "Rm9v",
	},
	{
		plain:   "test",
		encoded: "dGVzdA==",
	},
	{
		plain:   string([]byte{0}),
		encoded: "AA==",
	},
	{
		plain:   "some string with ünicødë",
		encoded: "c29tZSBzdHJpbmcgd2l0aCDDvG5pY8O4ZMOr",
	},
}

func TestEncode(t *testing.T) {
	for _, test := range testVectors {
		assert.Equal(t, test.encoded, string(base64.Encode([]byte(test.plain))))
	}
}

func TestEncodeToString(t *testing.T) {
	for _, test := range testVectors {
		assert.Equal(t, test.encoded, base64.EncodeToString([]byte(test.plain)))
	}
}

func TestDecode(t *testing.T) {
	for _, test := range testVectors {
		decoded, err := base64.Decode([]byte(test.encoded))
		assert.NoError(t, err)
		assert.Equal(t, test.plain, string(decoded))
	}
}

func TestDecodeString(t *testing.T) {
	for _, test := range testVectors {
		decoded, err := base64.DecodeString(test.encoded)
		assert.NoError(t, err)
		assert.Equal(t, test.plain, string(decoded))
	}
}
