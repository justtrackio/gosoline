package stream

const EncodingJson = "application/json"

var defaultEncoding = EncodingJson

func WithDefaultEncoding(encoding string) {
	defaultEncoding = encoding
}
