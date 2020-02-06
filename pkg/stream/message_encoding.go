package stream

import (
	"context"
	"fmt"
)

type EncodeHandler interface {
	Encode(ctx context.Context, attributes map[string]interface{}) (context.Context, map[string]interface{}, error)
	Decode(ctx context.Context, attributes map[string]interface{}) (context.Context, error)
}

type MessageEncoderConfig struct {
	Encoding       string
	Compression    string
	EncodeHandlers []EncodeHandler
}

type MessageEncoder struct {
	encoding       string
	compression    string
	encodeHandlers []EncodeHandler
}

func NewMessageEncoder(config *MessageEncoderConfig) *MessageEncoder {
	if config.Encoding == "" {
		config.Encoding = defaultMessageBodyEncoding
	}

	return &MessageEncoder{
		encoding:       config.Encoding,
		compression:    config.Compression,
		encodeHandlers: config.EncodeHandlers,
	}
}

func (e *MessageEncoder) Encode(ctx context.Context, data interface{}, attributeSets ...map[string]interface{}) (*Message, error) {
	var err error
	var msg *Message
	var attributes map[string]interface{}

	if msg, err = e.encodeBody(data); err != nil {
		return nil, err
	}

	if attributes, err = e.flattenAttributes(attributeSets); err != nil {
		return nil, err
	}

	for _, handler := range e.encodeHandlers {
		if ctx, attributes, err = handler.Encode(ctx, attributes); err != nil {
			return nil, fmt.Errorf("can not apply encoding handler on message: %w", err)
		}
	}

	for k, v := range attributes {
		if _, ok := msg.Attributes[k]; ok {
			return nil, fmt.Errorf("duplicate attribute %s on message", k)
		}

		msg.Attributes[k] = v
	}

	return msg, nil
}

func (e *MessageEncoder) encodeBody(data interface{}) (*Message, error) {
	if e.encoding == "" {
		return nil, fmt.Errorf("no encoding provided to encode message")
	}

	encoder, ok := messageBodyEncoders[e.encoding]

	if !ok {
		return nil, fmt.Errorf("there is no message body encoder available for encoding %s", e.encoding)
	}

	body, err := encoder.Encode(data)

	if err != nil {
		return nil, fmt.Errorf("can not encode message body with encoding %s: %w", e.encoding, err)
	}

	msg := &Message{
		Attributes: map[string]interface{}{
			AttributeEncoding: e.encoding,
		},
		Body: body,
	}

	return msg, nil
}

func (e *MessageEncoder) flattenAttributes(attributeSets []map[string]interface{}) (map[string]interface{}, error) {
	attributes := make(map[string]interface{})

	for _, set := range attributeSets {
		for k, v := range set {
			if _, ok := attributes[k]; ok {
				return nil, fmt.Errorf("duplicate attribute %s on message", k)
			}

			attributes[k] = v
		}
	}

	return attributes, nil
}

func (e *MessageEncoder) Decode(ctx context.Context, msg *Message, out interface{}) (context.Context, map[string]interface{}, error) {
	return nil, nil, nil
}
