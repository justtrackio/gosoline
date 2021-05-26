package uuid

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"strings"
)

func FromBytes(bytes []byte) (string, error) {
	s := BytesToHex(bytes)
	result, err := mdl.UuidWithDashes(&s)

	if err != nil {
		return "", err
	}

	return *result, nil
}

func ToBytes(uuid string) ([]byte, error) {
	s := strings.ReplaceAll(uuid, "-", "")

	if len(s) != 32 {
		return nil, fmt.Errorf("the uuid should be exactly 32 bytes long, but was: %d", len(s))
	}

	return HexToBytes(s)
}

func BytesToHex(bytes []byte) string {
	result := make([]byte, len(bytes)*2)

	for i, b := range bytes {
		result[i*2] = encodeHex(b >> 4)
		result[i*2+1] = encodeHex(b & 0xF)
	}

	return string(result)
}

func HexToBytes(s string) ([]byte, error) {
	if len(s)&1 != 0 {
		return nil, fmt.Errorf("expected even number of characters, got %d", len(s))
	}

	result := make([]byte, len(s)>>1)

	for i := 0; i < len(result); i++ {
		upper := parseHexChar(s[i*2])
		lower := parseHexChar(s[i*2+1])
		if upper < 0 || lower < 0 {
			return nil, fmt.Errorf("invalid byte at position %d: %s", i, s[i*2:i*2+2])
		}

		result[i] = byte((upper << 4) | lower)
	}

	return result, nil
}

func encodeHex(b byte) byte {
	if b < 10 {
		return b + '0'
	}
	return b - 10 + 'a'
}

func parseHexChar(c byte) int {
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return int(c - '0')
	case 'a', 'b', 'c', 'd', 'e', 'f':
		return int(c - 'a' + 10)
	case 'A', 'B', 'C', 'D', 'E', 'F':
		return int(c - 'A' + 10)
	default:
		return -1
	}
}
