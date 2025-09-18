package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"google.golang.org/protobuf/proto"
)

type base64LayeredProtobufEncoder struct{}

func NewBase64LayeredProtobufEncoder() MessageBodyEncoder {
	return base64LayeredProtobufEncoder{}
}

func (e base64LayeredProtobufEncoder) Encode(data any) ([]byte, error) {
	msg, ok := data.(ProtobufEncodable)

	if !ok {
		return nil, fmt.Errorf("%T does not implement ProtobufEncodable", data)
	}

	protoMsg, err := msg.ToMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to construct protobuf message: %w", err)
	}

	bytes, err := proto.Marshal(protoMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf message: %w", err)
	}

	// why do we need an extra layer of base64 here? Because, unlike JSON, protobuf makes use of all possible byte
	// values (because it embeds byte strings without encoding and is in general a binary format) and therefore we
	// need to encode it like that to ensure we can use the result in a json string.
	return base64.Encode(bytes), nil
}

func (e base64LayeredProtobufEncoder) Decode(data64 []byte, out any) error {
	msg, ok := out.(ProtobufEncodable)

	if !ok {
		return fmt.Errorf("%T does not implement ProtobufEncodable", out)
	}

	data, err := base64.Decode(data64)
	if err != nil {
		return fmt.Errorf("failed to decode protobuf base64 layer: %w", err)
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
