package cfg_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
)

type (
	Encoding    string
	Compression string
)

const (
	EncodingJson    Encoding    = "json"
	EncodingHtml    Encoding    = "html"
	EncodingPng     Encoding    = "png"
	CompressionGzip Compression = "gzip"
	CompressionNone Compression = "none"
)

type TestSettings struct {
	Encoding                  Encoding                                  `cfg:"encoding"`
	FallbackEncoding          Encoding                                  `cfg:"fallback_encoding" default:"json"`
	Compression               Compression                               `cfg:"compression" validate:"oneof=gzip none"`
	FallbackCompression       Compression                               `cfg:"fallback_compression" default:"none" validate:"oneof=gzip none"`
	SupportedCompression      []Compression                             `cfg:"supported_compression" validate:"min=1"`
	EncodingCompression       map[Encoding]Compression                  `cfg:"encoding_compression"`
	SupportedCompressions     map[Encoding][]Compression                `cfg:"supported_compressions"`
	AllEncodings              []Encoding                                `cfg:"all_encodings"`
	OkayIDontKnowANameForThis map[Encoding][]map[Encoding][]Compression `cfg:"name_this"`
}

type ApiSettings struct {
	Port string `cfg:"port"`
	Mode string `cfg:"mode"`
}

func TestConfigStringsTest(t *testing.T) {
	config := cfg.New()
	err := config.Option(cfg.WithConfigFile("./testdata/config.strings.test.yml", "yml"))
	assert.NoError(t, err)

	settings := &TestSettings{}
	config.UnmarshalKey("test", settings)

	assert.Equal(t, &TestSettings{
		Encoding:            EncodingJson,
		FallbackEncoding:    EncodingJson,
		Compression:         CompressionGzip,
		FallbackCompression: CompressionNone,
		SupportedCompression: []Compression{
			CompressionGzip,
			CompressionNone,
		},
		EncodingCompression: map[Encoding]Compression{
			EncodingJson: CompressionGzip,
			EncodingHtml: CompressionGzip,
			EncodingPng:  CompressionNone,
		},
		SupportedCompressions: map[Encoding][]Compression{
			EncodingJson: {CompressionGzip, CompressionNone},
			EncodingHtml: {CompressionGzip},
			EncodingPng:  {CompressionNone},
		},
		AllEncodings: []Encoding{
			EncodingJson,
			EncodingHtml,
			EncodingPng,
		},
		OkayIDontKnowANameForThis: map[Encoding][]map[Encoding][]Compression{
			EncodingJson: {
				{
					EncodingHtml: {
						CompressionGzip,
						CompressionNone,
					},
					EncodingPng: {
						CompressionNone,
					},
				},
				{
					EncodingHtml: {},
					EncodingJson: {
						CompressionNone,
					},
				},
			},
			EncodingHtml: {
				{
					EncodingJson: {},
				},
				{
					EncodingHtml: {},
					EncodingJson: {
						CompressionGzip,
						CompressionGzip,
					},
				},
			},
		},
	}, settings)
}

func TestConfigStringConversionTest(t *testing.T) {
	config := cfg.New()
	err := config.Option(cfg.WithConfigFile("./testdata/config.strings.test.yml", "yml"))
	assert.NoError(t, err)

	settings := &ApiSettings{}
	config.UnmarshalKey("api", settings)

	assert.Equal(t, &ApiSettings{
		Port: "80",
		Mode: "release",
	}, settings)
}
