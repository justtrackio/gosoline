package stream

import (
	"context"
	"fmt"
)

type EncodeHandler interface {
	Encode(ctx context.Context, attributes map[string]interface{}) (context.Context, map[string]interface{}, error)
	Decode(ctx context.Context, attributes map[string]interface{}) (context.Context, map[string]interface{}, error)
}

var defaultEncodeHandlers = make([]EncodeHandler, 0)

func AddDefaultEncodeHandler(handler EncodeHandler) {
	defaultEncodeHandlers = append(defaultEncodeHandlers, handler)
}

type MessageEncoderSettings struct {
	Encoding       string
	Compression    string
	EncodeHandlers []EncodeHandler
}

type MessageEncoder interface {
	Encode(ctx context.Context, data interface{}, attributeSets ...map[string]interface{}) (*Message, error)
	Decode(ctx context.Context, msg *Message, out interface{}) (context.Context, map[string]interface{}, error)
}

type messageEncoder struct {
	encoding       string
	compression    string
	encodeHandlers []EncodeHandler
}

func NewMessageEncoder(config *MessageEncoderSettings) *messageEncoder {
	if config.Encoding == "" {
		config.Encoding = defaultMessageBodyEncoding
	}

	if config.Compression == "" {
		config.Compression = "none"
	}

	if len(config.EncodeHandlers) == 0 {
		config.EncodeHandlers = defaultEncodeHandlers
	}

	return &messageEncoder{
		encoding:       config.Encoding,
		compression:    config.Compression,
		encodeHandlers: config.EncodeHandlers,
	}
}

func (e *messageEncoder) Encode(ctx context.Context, data interface{}, attributeSets ...map[string]interface{}) (*Message, error) {
	var err error
	var msg *Message
	var attributes map[string]interface{}

	if msg, err = e.encodeBody(data); err != nil {
		return nil, err
	}

	if attributes, err = e.mergeAttributes(attributeSets); err != nil {
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

func (e *messageEncoder) encodeBody(data interface{}) (*Message, error) {
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

func (e *messageEncoder) mergeAttributes(attributeSets []map[string]interface{}) (map[string]interface{}, error) {
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

func (e *messageEncoder) Decode(ctx context.Context, msg *Message, out interface{}) (context.Context, map[string]interface{}, error) {
	var err error
	var attributes map[string]interface{}

	if attributes, err = e.decodeBody(msg, out); err != nil {
		return ctx, attributes, fmt.Errorf("can not decode message body: %w", err)
	}

	for _, handler := range e.encodeHandlers {
		if ctx, attributes, err = handler.Decode(ctx, attributes); err != nil {
			return ctx, attributes, fmt.Errorf("can not apply encoding handler on message: %w", err)
		}
	}

	return ctx, attributes, nil
}

func (e *messageEncoder) decodeBody(msg *Message, out interface{}) (map[string]interface{}, error) {
	attributes := msg.Attributes
	encoding := e.encoding

	if attrEncoding, ok := attributes[AttributeEncoding]; ok {
		if _, ok := attrEncoding.(string); !ok {
			return attributes, fmt.Errorf("the encoding set in the message attributes should be of type string")
		}

		encoding = attrEncoding.(string)
	}

	encoder, ok := messageBodyEncoders[encoding]

	if !ok {
		return attributes, fmt.Errorf("there is no message body decoder available for encoding %s", encoding)
	}

	err := encoder.Decode(msg.Body, out)

	if err != nil {
		return attributes, fmt.Errorf("can not decode message body with encoding %s: %w", encoding, err)
	}

	delete(attributes, AttributeEncoding)

	return attributes, nil
}
