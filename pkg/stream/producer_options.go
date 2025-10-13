package stream

type producerOptions struct {
	encodeHandlers []EncodeHandler
	schemaSettings *SchemaSettings
}
type ProducerOption func(p *producerOptions)

func WithEncodeHandlers(encodeHandlers []EncodeHandler) ProducerOption {
	return func(p *producerOptions) {
		p.encodeHandlers = encodeHandlers
	}
}

func WithSchemaSettings(schemaSettings SchemaSettings) ProducerOption {
	return func(p *producerOptions) {
		p.schemaSettings = &schemaSettings
	}
}
