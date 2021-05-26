package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/encoding/base64"
)

type EncodeHandler interface {
	Encode(ctx context.Context, data interface{}, attributes map[string]interface{}) (context.Context, map[string]interface{}, error)
	Decode(ctx context.Context, data interface{}, attributes map[string]interface{}) (context.Context, map[string]interface{}, error)
}

var defaultEncodeHandlers = make([]EncodeHandler, 0)

func AddDefaultEncodeHandler(handler EncodeHandler) {
	defaultEncodeHandlers = append(defaultEncodeHandlers, handler)
}

type MessageEncoderSettings struct {
	Encoding       EncodingType
	Compression    CompressionType
	EncodeHandlers []EncodeHandler
}

//go:generate mockery -name MessageEncoder
type MessageEncoder interface {
	Encode(ctx context.Context, data interface{}, attributeSets ...map[string]interface{}) (*Message, error)
	Decode(ctx context.Context, msg *Message, out interface{}) (context.Context, map[string]interface{}, error)
}

type messageEncoder struct {
	encoding       EncodingType
	compression    CompressionType
	encodeHandlers []EncodeHandler
}

func NewMessageEncoder(config *MessageEncoderSettings) *messageEncoder {
	if config.Encoding == "" {
		config.Encoding = defaultMessageBodyEncoding
	}

	if config.Compression == "" {
		config.Compression = CompressionNone
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
	var body []byte
	attributes := make(map[string]interface{})

	if body, err = e.encodeBody(attributes, data); err != nil {
		return nil, fmt.Errorf("could not encode message body: %w", err)
	}

	if body, err = e.compressBody(attributes, body); err != nil {
		return nil, fmt.Errorf("could not compress message body: %w", err)
	}

	if attributes, err = e.mergeAttributes(attributes, attributeSets); err != nil {
		return nil, err
	}

	for _, handler := range e.encodeHandlers {
		if ctx, attributes, err = handler.Encode(ctx, data, attributes); err != nil {
			return nil, fmt.Errorf("can not apply encoding handler on message: %w", err)
		}
	}

	msg := &Message{
		Attributes: attributes,
		Body:       string(body),
	}

	return msg, nil
}

func (e *messageEncoder) encodeBody(attributes map[string]interface{}, data interface{}) ([]byte, error) {
	body, err := EncodeMessage(e.encoding, data)

	if err != nil {
		return nil, err
	}

	attributes[AttributeEncoding] = e.encoding

	return body, nil
}

func (e *messageEncoder) compressBody(attributes map[string]interface{}, body []byte) ([]byte, error) {
	if e.compression == CompressionNone {
		return body, nil
	}

	compressed, err := CompressMessage(e.compression, body)

	if err != nil {
		return nil, err
	}

	compressedBase64 := base64.Encode(compressed)
	attributes[AttributeCompression] = e.compression

	return compressedBase64, nil
}

func (e *messageEncoder) mergeAttributes(attributes map[string]interface{}, attributeSets []map[string]interface{}) (map[string]interface{}, error) {
	for _, set := range attributeSets {
		for k, v := range set {
			if _, ok := attributes[k]; ok {
				return nil, fmt.Errorf("duplicate attribute '%s' on message", k)
			}

			attributes[k] = v
		}
	}

	return attributes, nil
}

func (e *messageEncoder) Decode(ctx context.Context, msg *Message, out interface{}) (context.Context, map[string]interface{}, error) {
	var err error
	var body []byte

	attributes := msg.Attributes
	body = []byte(msg.Body)

	if body, err = e.decompressBody(attributes, body); err != nil {
		return ctx, attributes, err
	}

	if err = e.decodeBody(attributes, body, out); err != nil {
		return ctx, attributes, fmt.Errorf("can not decode message body: %w", err)
	}

	for _, handler := range e.encodeHandlers {
		if ctx, attributes, err = handler.Decode(ctx, out, attributes); err != nil {
			return ctx, attributes, fmt.Errorf("can not apply encoding handler on message: %w", err)
		}
	}

	return ctx, attributes, nil
}

func (e *messageEncoder) decompressBody(attributes map[string]interface{}, body []byte) ([]byte, error) {
	compression, err := GetCompressionAttribute(attributes)

	if err != nil {
		return body, err
	}

	if compression == nil {
		return body, nil
	}

	base64Decoded, err := base64.Decode(body)

	if err != nil {
		return nil, fmt.Errorf("can not base64 decode the body: %w", err)
	}

	decompressed, err := DecompressMessage(*compression, base64Decoded)

	if err != nil {
		return nil, err
	}

	delete(attributes, AttributeCompression)

	return decompressed, nil
}

func (e *messageEncoder) decodeBody(attributes map[string]interface{}, body []byte, out interface{}) error {
	encoding := e.encoding

	if attrEncoding, err := GetEncodingAttribute(attributes); err != nil {
		return err
	} else if attrEncoding != nil {
		encoding = *attrEncoding
	}

	if err := DecodeMessage(encoding, body, out); err != nil {
		return err
	}

	delete(attributes, AttributeEncoding)

	return nil
}
