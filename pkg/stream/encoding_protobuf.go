package stream

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

type ProtobufEncodable interface {
	ToMessage() (proto.Message, error)
	EmptyMessage() proto.Message
	FromMessage(message proto.Message) error
}

type protobufEncoder struct{}

func NewProtobufEncoder() MessageBodyEncoder {
	return protobufEncoder{}
}

func (e protobufEncoder) Encode(data any) ([]byte, error) {
	msg, ok := data.(ProtobufEncodable)

	if !ok {
		return nil, fmt.Errorf("%T does not implement ProtobufEncodable", data)
	}

	protoMsg, err := msg.ToMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to construct protobuf message: %w", err)
	}

	return proto.Marshal(protoMsg)
}

func (e protobufEncoder) Decode(data []byte, out any) error {
	msg, ok := out.(ProtobufEncodable)

	if !ok {
		return fmt.Errorf("%T does not implement ProtobufEncodable", out)
	}

	// create an empty message from an empty struct
	protoMsg := msg.EmptyMessage()

	if err := proto.Unmarshal(data, protoMsg); err != nil {
		return fmt.Errorf("failed to decode protobuf message: %w", err)
	}

	if err := msg.FromMessage(protoMsg); err != nil {
		return fmt.Errorf("failed to convert protobuf message: %w", err)
	}

	return nil
}
