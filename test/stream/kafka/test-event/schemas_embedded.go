package test_event

import _ "embed"

//go:embed TestEvent.avsc
var SchemaAvro string

//go:embed TestEvent.proto
var SchemaProto string

//go:embed TestEvent.json
var SchemaJson string
