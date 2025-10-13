package stream

type SchemaSettingsWithEncoding struct {
	Subject              string
	Schema               string
	Encoding             EncodingType
	ProtobufMessageIndex []int
	Model                any
}

type SchemaSettings struct {
	Subject              string
	Schema               string
	ProtobufMessageIndex []int
	Model                any
}

func (s SchemaSettings) WithEncoding(encoding EncodingType) SchemaSettingsWithEncoding {
	return SchemaSettingsWithEncoding{
		Subject:              s.Subject,
		Schema:               s.Schema,
		Encoding:             encoding,
		ProtobufMessageIndex: s.ProtobufMessageIndex,
		Model:                s.Model,
	}
}
