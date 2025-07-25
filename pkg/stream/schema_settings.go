package stream

type SchemaSettings struct {
	Subject              string
	Schema               string
	ProtobufMessageIndex []int
	Model                any
}

type SchemaSettingsWithEncoding struct {
	Subject              string
	Schema               string
	Encoding             EncodingType
	ProtobufMessageIndex []int
	Model                any
}
