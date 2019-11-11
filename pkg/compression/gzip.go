package compression

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Gzip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err := gz.Write(data)
	if err != nil {
		return nil, err
	}

	if err = gz.Flush(); err != nil {
		return nil, err
	}

	if err = gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func GzipString(data string) ([]byte, error) {
	return Gzip([]byte(data))
}

func Gunzip(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)

	var r io.Reader
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	return resB.Bytes(), nil
}

func GunzipToString(data []byte) (string, error) {
	uncompressedData, err := Gunzip(data)

	return string(uncompressedData), err
}
