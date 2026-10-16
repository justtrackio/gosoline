// Package base64 encodes and decodes bytes using the standard base64 alphabet.
//
// Deprecated: Use the standard library's encoding/base64 package instead.
package base64

import "encoding/base64"

// Encode returns the base64 encoding of src using the standard alphabet.
//
// Deprecated: Use encoding/base64.StdEncoding.AppendEncode([]byte{}, src) instead (Go 1.22).
func Encode(src []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(buf, src)

	return buf
}

// EncodeToString returns the base64 encoding of v using the standard alphabet.
//
// Deprecated: Use encoding/base64.StdEncoding.EncodeToString instead.
func EncodeToString(v []byte) string {
	return base64.StdEncoding.EncodeToString(v)
}

// Decode decodes src using the standard base64 alphabet.
//
// Deprecated: Use encoding/base64.StdEncoding.AppendDecode([]byte{}, src) instead (Go 1.22).
func Decode(src []byte) ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(buf, src)

	return buf[0:n], err
}

// DecodeString decodes v using the standard base64 alphabet.
//
// Deprecated: Use encoding/base64.StdEncoding.DecodeString instead.
func DecodeString(v string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(v)
}
