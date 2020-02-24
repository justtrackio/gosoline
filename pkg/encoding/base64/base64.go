package base64

import "encoding/base64"

func Encode(src []byte) []byte {
	buf := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(buf, src)

	return buf
}

func EncodeToString(v []byte) string {
	return base64.StdEncoding.EncodeToString(v)
}

func Decode(src []byte) ([]byte, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	_, err := base64.StdEncoding.Decode(buf, src)

	return buf, err
}

func DecodeString(v string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(v)
}
