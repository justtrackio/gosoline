package uuid_test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestFromBytes(t *testing.T) {
	type testCase struct {
		input   []byte
		decoded string
		err     error
	}

	for name, test := range map[string]testCase{
		"empty": {
			input:   []byte{},
			decoded: "",
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 0"),
		},
		"zeroUuid": {
			input:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			decoded: "00000000-0000-0000-0000-000000000000",
			err:     nil,
		},
		"validUuid": {
			input:   []byte{0, 1, 2, 4, 8, 16, 32, 64, 128, 255, 31, 7, 3, 1, 0, 1},
			decoded: "00010204-0810-2040-80ff-1f0703010001",
			err:     nil,
		},
		"shortUuid": {
			input:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			decoded: "",
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 30"),
		},
		"longUuid": {
			input:   []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			decoded: "",
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 34"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := uuid.FromBytes(test.input)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.decoded, decoded)
		})
	}
}

func TestToBytes(t *testing.T) {
	type testCase struct {
		input   string
		encoded []byte
		err     error
	}

	for name, test := range map[string]testCase{
		"empty": {
			input:   "",
			encoded: nil,
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 0"),
		},
		"zeroUuid": {
			input:   "00000000-0000-0000-0000-000000000000",
			encoded: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			err:     nil,
		},
		"formattedLikeUuid": {
			input:   "zzaayyxx-aahh-55ag-8gaq-a55hsbyjs35h",
			encoded: nil,
			err:     fmt.Errorf("invalid byte at position 0: zz"),
		},
		"validUuid": {
			input:   "9cb93b4b-cba9-49e3-b6fa-50c51ec9354c",
			encoded: []byte{0x9c, 0xb9, 0x3b, 0x4b, 0xcb, 0xa9, 0x49, 0xe3, 0xb6, 0xfa, 0x50, 0xc5, 0x1e, 0xc9, 0x35, 0x4c},
			err:     nil,
		},
		"shortUuid": {
			input:   "9cb93b4b-cba9-49e3-b6fa-50c51ec935",
			encoded: nil,
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 30"),
		},
		"longUuid": {
			input:   "9cb93b4b-cba9-49e3-b6fa-50c51ec9354c and some more",
			encoded: nil,
			err:     fmt.Errorf("the uuid should be exactly 32 bytes long, but was: 46"),
		},
		"missingHyphens": {
			input:   "9cb93b4bcba949e3b6fa50c51ec9354c",
			encoded: []byte{0x9c, 0xb9, 0x3b, 0x4b, 0xcb, 0xa9, 0x49, 0xe3, 0xb6, 0xfa, 0x50, 0xc5, 0x1e, 0xc9, 0x35, 0x4c},
			err:     nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := uuid.ToBytes(test.input)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.encoded, encoded)
		})
	}
}

func TestToBytesFromBytes(t *testing.T) {
	for i := 0; i < 100; i++ {
		uuidString := uuid.New().NewV4()
		encoded, err := uuid.ToBytes(uuidString)
		assert.NoError(t, err)
		decoded, err := uuid.FromBytes(encoded)
		assert.NoError(t, err)
		assert.Equal(t, uuidString, decoded)
	}
}

func TestBytesToHexToBytes(t *testing.T) {
	for i := 0; i < 100; i++ {
		bytes := make([]byte, int(rand.Uint32()%256))
		for j := range bytes {
			bytes[j] = byte(rand.Int())
		}
		encoded := uuid.BytesToHex(bytes)
		decoded, err := uuid.HexToBytes(encoded)
		assert.NoError(t, err)
		assert.Equal(t, bytes, decoded)
	}
}

func TestBytesToHex(t *testing.T) {
	assert.Equal(t, "", uuid.BytesToHex([]byte{}))
	assert.Equal(t, "05", uuid.BytesToHex([]byte{5}))
	assert.Equal(t, "ff", uuid.BytesToHex([]byte{255}))
	assert.Equal(t, "00000000", uuid.BytesToHex([]byte{0, 0, 0, 0}))
	assert.Equal(t, "10012002a00aa22aabba", uuid.BytesToHex([]byte{0x10, 0x01, 0x20, 0x02, 0xa0, 0x0a, 0xa2, 0x2a, 0xab, 0xba}))
}

func TestHexToBytes(t *testing.T) {
	type testCase struct {
		input   string
		decoded []byte
		err     error
	}

	for name, test := range map[string]testCase{
		"empty": {
			input:   "",
			decoded: []byte{},
			err:     nil,
		},
		"zeroBytes": {
			input:   "00000000000000000000000000000000",
			decoded: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			err:     nil,
		},
		"withJunk": {
			input:   "abcdefgh",
			decoded: nil,
			err:     fmt.Errorf("invalid byte at position 3: gh"),
		},
		"validData": {
			input:   "10012002a00aa22aabba",
			decoded: []byte{0x10, 0x01, 0x20, 0x02, 0xa0, 0x0a, 0xa2, 0x2a, 0xab, 0xba},
			err:     nil,
		},
		"missingCharacter": {
			input:   "10012002a00aa22aabb",
			decoded: nil,
			err:     fmt.Errorf("expected even number of characters, got 19"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			decoded, err := uuid.HexToBytes(test.input)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.decoded, decoded)
		})
	}
}
