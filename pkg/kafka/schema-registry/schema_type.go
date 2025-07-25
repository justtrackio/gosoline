package schema_registry

type SchemaType string

const (
	Avro     SchemaType = "avro"
	Json     SchemaType = "json"
	Protobuf SchemaType = "protobuf"
)
