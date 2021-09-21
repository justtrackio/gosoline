package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"google.golang.org/protobuf/proto"
)

type EncodingType string

const (
	EncodingJson     EncodingType = "application/json"
	EncodingProtobuf EncodingType = "application/x-protobuf"
)

func (s EncodingType) String() string {
	return string(s)
}

var _ fmt.Stringer = EncodingType("")

type ProtobufEncodable interface {
	ToMessage() (proto.Message, error)
	EmptyMessage() proto.Message
	FromMessage(message proto.Message) error
}

var defaultMessageBodyEncoding = EncodingJson

func WithDefaultMessageBodyEncoding(encoding EncodingType) {
	defaultMessageBodyEncoding = encoding
}

type MessageBodyEncoder interface {
	Encode(data interface{}) ([]byte, error)
	Decode(data []byte, out interface{}) error
}

var messageBodyEncoders = map[EncodingType]MessageBodyEncoder{
	EncodingJson:     new(jsonEncoder),
	EncodingProtobuf: new(protobufEncoder),
}

func AddMessageBodyEncoder(encoding EncodingType, encoder MessageBodyEncoder) {
	messageBodyEncoders[encoding] = encoder
}

func EncodeMessage(encoding EncodingType, data interface{}) ([]byte, error) {
	if encoding == "" {
		return nil, fmt.Errorf("no encoding provided to encode message")
	}

	encoder, ok := messageBodyEncoders[encoding]

	if !ok {
		return nil, fmt.Errorf("there is no message body encoder available for encoding '%s'", encoding)
	}

	body, err := encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("can not encode message body with encoding '%s': %w", encoding, err)
	}

	return body, nil
}

func DecodeMessage(encoding EncodingType, data []byte, out interface{}) error {
	encoder, ok := messageBodyEncoders[encoding]

	if !ok {
		return fmt.Errorf("there is no message body decoder available for encoding '%s'", encoding)
	}

	err := encoder.Decode(data, out)
	if err != nil {
		return fmt.Errorf("can not decode message body with encoding '%s': %w", encoding, err)
	}

	return nil
}

type jsonEncoder struct{}

func NewJsonEncoder() MessageBodyEncoder {
	return jsonEncoder{}
}

func (e jsonEncoder) Encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (e jsonEncoder) Decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

type protobufEncoder struct{}

func NewProtobufEncoder() MessageBodyEncoder {
	return protobufEncoder{}
}

func (e protobufEncoder) Encode(data interface{}) ([]byte, error) {
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

func (e protobufEncoder) Decode(data64 []byte, out interface{}) error {
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
