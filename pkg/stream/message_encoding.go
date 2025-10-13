package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type EncodeHandler interface {
	Encode(ctx context.Context, data any, attributes map[string]string) (context.Context, map[string]string, error)
	Decode(ctx context.Context, data any, attributes map[string]string) (context.Context, map[string]string, error)
}

var defaultEncodeHandlers = make([]EncodeHandler, 0)

func AddDefaultEncodeHandler(handler EncodeHandler) {
	defaultEncodeHandlers = append(defaultEncodeHandlers, handler)
}

type MessageEncoderSettings struct {
	Encoding        EncodingType
	Compression     CompressionType
	EncodeHandlers  []EncodeHandler
	ExternalEncoder MessageBodyEncoder
}

//go:generate go run github.com/vektra/mockery/v2 --name MessageEncoder
type MessageEncoder interface {
	Encode(ctx context.Context, data any, attributeSets ...map[string]string) (*Message, error)
	Decode(ctx context.Context, msg *Message, out any) (context.Context, map[string]string, error)
}

type messageEncoder struct {
	encoding        EncodingType
	compression     CompressionType
	encodeHandlers  []EncodeHandler
	externalEncoder MessageBodyEncoder
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
		encoding:        config.Encoding,
		compression:     config.Compression,
		encodeHandlers:  config.EncodeHandlers,
		externalEncoder: config.ExternalEncoder,
	}
}

func (e *messageEncoder) Encode(ctx context.Context, data any, attributeSets ...map[string]string) (*Message, error) {
	var err error
	var body []byte
	attributes := make(map[string]string)

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

func (e *messageEncoder) encodeBody(attributes map[string]string, data any) ([]byte, error) {
	if e.externalEncoder != nil {
		body, err := e.externalEncoder.Encode(data)
		if err != nil {
			return nil, fmt.Errorf("external encoding failed: %w", err)
		}

		return body, nil
	}

	body, err := EncodeMessage(e.encoding, data)
	if err != nil {
		return nil, err
	}

	attributes[AttributeEncoding] = e.encoding.String()

	return body, nil
}

func (e *messageEncoder) compressBody(attributes map[string]string, body []byte) ([]byte, error) {
	if e.compression == CompressionNone {
		return body, nil
	}

	compressed, err := CompressMessage(e.compression, body)
	if err != nil {
		return nil, err
	}

	compressedBase64 := base64.Encode(compressed)
	attributes[AttributeCompression] = e.compression.String()

	return compressedBase64, nil
}

func (e *messageEncoder) mergeAttributes(attributes map[string]string, attributeSets []map[string]string) (map[string]string, error) {
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

func (e *messageEncoder) Decode(ctx context.Context, msg *Message, out any) (context.Context, map[string]string, error) {
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

func (e *messageEncoder) decompressBody(attributes map[string]string, body []byte) ([]byte, error) {
	compression := GetCompressionAttribute(attributes)

	if compression == nil {
		return body, nil
	}

	base64Decoded, err := base64.Decode(body)
	if err != nil {
		return nil, fmt.Errorf("can not base64 decode the body: %w", err)
	}

	return DecompressMessage(*compression, base64Decoded)
}

func (e *messageEncoder) decodeBody(attributes map[string]string, body []byte, out any) error {
	if e.externalEncoder != nil {
		if err := e.externalEncoder.Decode(body, out); err != nil {
			return fmt.Errorf("external decoding failed: %w", err)
		}

		return nil
	}

	encoding := mdl.Unbox(GetEncodingAttribute(attributes), e.encoding)

	return DecodeMessage(encoding, body, out)
}
