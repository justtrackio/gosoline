package test_event

import (
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/test/stream/kafka/test-event/protobuf"
	"google.golang.org/protobuf/proto"
)

var _ stream.ProtobufEncodable = &TestEvent{}

//go:generate protoc --go_out=./protobuf --go_opt=paths=source_relative TestEvent.proto
type TestEvent struct {
	Id   int    `avro:"id" json:"id"`
	Name string `avro:"name" json:"name"`
}

func (e *TestEvent) EmptyMessage() proto.Message {
	return &protobuf.TestEvent{}
}

func (e *TestEvent) ToMessage() (proto.Message, error) {
	return &protobuf.TestEvent{
		Id:   int32(e.Id),
		Name: e.Name,
	}, nil
}

func (e *TestEvent) FromMessage(message proto.Message) error {
	msg := message.(*protobuf.TestEvent)

	*e = TestEvent{
		Id:   int(msg.Id),
		Name: msg.Name,
	}

	return nil
}
