package stream

type SchemaSettingsWithEncoding struct {
	Subject              string
	Schema               string
	Encoding             EncodingType
	AutoRegister         bool
	ProtobufMessageIndex []int
	Model                any
}

type SchemaSettings struct {
	Subject              string
	Schema               string
	AutoRegister         bool
	ProtobufMessageIndex []int
	Model                any
}

func (s SchemaSettings) WithEncoding(encoding EncodingType) SchemaSettingsWithEncoding {
	return SchemaSettingsWithEncoding{
		Subject:              s.Subject,
		Schema:               s.Schema,
		Encoding:             encoding,
		AutoRegister:         s.AutoRegister,
		ProtobufMessageIndex: s.ProtobufMessageIndex,
		Model:                s.Model,
	}
}
