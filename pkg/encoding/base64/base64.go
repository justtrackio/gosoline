package base64

import b64 "encoding/base64"

func Encode(v []byte) string {
	return b64.StdEncoding.EncodeToString(v)
}

func DecodeString(v string) ([]byte, error) {
	return b64.StdEncoding.DecodeString(v)
}
