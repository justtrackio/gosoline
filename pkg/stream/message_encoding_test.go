package stream_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/suite"
)

type encodingTestStruct struct {
	Id        int       `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type brokenEncodeHandler struct{}

func (b brokenEncodeHandler) Encode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	return ctx, attributes, fmt.Errorf("encode handler encode error")
}

func (b brokenEncodeHandler) Decode(ctx context.Context, _ any, attributes map[string]string) (context.Context, map[string]string, error) {
	return ctx, attributes, fmt.Errorf("encode handler decode error")
}

type MessageEncoderSuite struct {
	suite.Suite
	clock clock.Clock
}

func (s *MessageEncoderSuite) SetupTest() {
	s.clock = clock.NewFakeClock()
}

func (s *MessageEncoderSuite) TestEncode() {
	// {"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}
	data := encodingTestStruct{
		Id:        3,
		Text:      "example",
		CreatedAt: s.clock.Now(),
	}

	tests := map[string]struct {
		encoding           stream.EncodingType
		compression        stream.CompressionType
		handlers           []stream.EncodeHandler
		attributes         map[string]string
		expectedError      string
		expectedBody       string
		expectedAttributes map[string]string
	}{
		"encoding_missing": {
			encoding:      "missing",
			expectedError: "could not encode message body: there is no message body encoder available for encoding 'missing'",
		},
		"compression_missing": {
			encoding:      stream.EncodingJson,
			compression:   "missing",
			expectedError: "could not compress message body: there is no compressor for compression 'missing'",
		},
		"attribute_duplicate": {
			encoding:    stream.EncodingJson,
			compression: stream.CompressionNone,
			attributes: map[string]string{
				stream.AttributeEncoding: "duplicate",
			},
			expectedError: "duplicate attribute 'encoding' on message",
		},
		"broken_handler": {
			encoding: stream.EncodingJson,
			handlers: []stream.EncodeHandler{
				new(brokenEncodeHandler),
			},
			expectedError: "can not apply encoding handler on message: encode handler encode error",
		},
		"json_uncompressed": {
			encoding:    stream.EncodingJson,
			compression: stream.CompressionNone,
			attributes: map[string]string{
				"attribute1": "5",
				"attribute2": "test",
			},
			expectedBody: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
			expectedAttributes: map[string]string{
				"attribute1":             "5",
				"attribute2":             "test",
				stream.AttributeEncoding: stream.EncodingJson.String(),
			},
		},
		"json_compressed": {
			encoding:    stream.EncodingJson,
			compression: stream.CompressionGZip,
			attributes: map[string]string{
				"attribute1": "5",
				"attribute2": "test",
			},
			expectedBody: `H4sIAAAAAAAA/6pWykxRsjLWUSpJrShRslJKrUjMLchJVdJRSi5KTSxJTXEEiRpaWpjoGoBQiIGBFRhFKdUCAgAA//9Q/bHSPAAAAA==`,
			expectedAttributes: map[string]string{
				"attribute1":                "5",
				"attribute2":                "test",
				stream.AttributeEncoding:    stream.EncodingJson.String(),
				stream.AttributeCompression: stream.CompressionGZip.String(),
			},
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
				Encoding:       tt.encoding,
				Compression:    tt.compression,
				EncodeHandlers: tt.handlers,
			})

			ctx := s.T().Context()
			msg, err := encoder.Encode(ctx, data, tt.attributes)

			if tt.expectedError != "" {
				s.EqualError(err, tt.expectedError)

				return
			}

			s.NoError(err)
			s.Equal(tt.expectedBody, msg.Body)
			s.Len(msg.Attributes, len(tt.expectedAttributes))

			for k, v := range tt.expectedAttributes {
				s.Contains(msg.Attributes, k)
				s.Equal(v, msg.Attributes[k])
			}
		})
	}
}

func (s *MessageEncoderSuite) TestDecode() {
	expectedStruct := &encodingTestStruct{
		Id:        3,
		Text:      "example",
		CreatedAt: s.clock.Now(),
	}

	tests := map[string]struct {
		handlers           []stream.EncodeHandler
		message            *stream.Message
		actualOutput       any
		expectedError      string
		expectedAttributes map[string]string
		expectedOutput     any
	}{
		"compression_missing": {
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeCompression: "missing",
					stream.AttributeEncoding:    stream.EncodingJson.String(),
				},
				Body: `H4sIAAAAAAAA/6pWykxRsjLWUSpJrShRslJKrUjMLchJVdJRSi5KTSxJTXEEiRpaWpjoGoBQiIGBFRhFKdUCAAAA//8BAAD//1D9sdI8AAAA`,
			},
			actualOutput:   &encodingTestStruct{},
			expectedError:  "there is no decompressor for compression 'missing'",
			expectedOutput: expectedStruct,
		},
		"encoding_missing": {
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeEncoding: "missing",
				},
				Body: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
			},
			actualOutput:   &encodingTestStruct{},
			expectedError:  "can not decode message body: there is no message body decoder available for encoding 'missing'",
			expectedOutput: expectedStruct,
		},
		"broken_handler": {
			handlers: []stream.EncodeHandler{
				new(brokenEncodeHandler),
			},
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeEncoding: stream.EncodingJson.String(),
				},
				Body: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
			},
			actualOutput:   &encodingTestStruct{},
			expectedError:  "can not apply encoding handler on message: encode handler decode error",
			expectedOutput: expectedStruct,
		},
		"json_uncompressed": {
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeEncoding: stream.EncodingJson.String(),
					"attribute1":             "5",
					"attribute2":             "test",
				},
				Body: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
			},
			actualOutput: &encodingTestStruct{},
			expectedAttributes: map[string]string{
				stream.AttributeEncoding: stream.EncodingJson.String(),
				"attribute1":             "5",
				"attribute2":             "test",
			},
			expectedOutput: expectedStruct,
		},
		"json_compressed": {
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeCompression: stream.CompressionGZip.String(),
					stream.AttributeEncoding:    stream.EncodingJson.String(),
				},
				Body: `H4sIAAAAAAAA/6pWykxRsjLWUSpJrShRslJKrUjMLchJVdJRSi5KTSxJTXEEiRpaWpjoGoBQiIGBFRhFKdUCAAAA//8BAAD//1D9sdI8AAAA`,
			},
			actualOutput: &encodingTestStruct{},
			expectedAttributes: map[string]string{
				stream.AttributeCompression: stream.CompressionGZip.String(),
				stream.AttributeEncoding:    stream.EncodingJson.String(),
			},
			expectedOutput: expectedStruct,
		},
		"json_into_msi": {
			message: &stream.Message{
				Attributes: map[string]string{
					stream.AttributeEncoding: stream.EncodingJson.String(),
					"attribute1":             "5",
					"attribute2":             "test",
				},
				Body: `{"id":3,"text":"example","createdAt":"1984-04-04T00:00:00Z"}`,
			},
			actualOutput: &map[string]any{},
			expectedAttributes: map[string]string{
				stream.AttributeEncoding: stream.EncodingJson.String(),
				"attribute1":             "5",
				"attribute2":             "test",
			},
			expectedOutput: &map[string]any{
				"id":        float64(3),
				"text":      "example",
				"createdAt": "1984-04-04T00:00:00Z",
			},
		},
	}

	for name, tt := range tests {
		s.Run(name, func() {
			encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
				EncodeHandlers: tt.handlers,
			})

			ctx := s.T().Context()
			_, attributes, err := encoder.Decode(ctx, tt.message, tt.actualOutput)

			if tt.expectedError != "" {
				s.EqualError(err, tt.expectedError)

				return
			}

			s.NoError(err)
			s.Equal(tt.expectedOutput, tt.actualOutput)
			s.Len(attributes, len(tt.expectedAttributes))

			for k, v := range tt.expectedAttributes {
				s.Contains(attributes, k)
				s.Equal(v, attributes[k])
			}
		})
	}
}

func TestMessageEncoderSuite(t *testing.T) {
	suite.Run(t, new(MessageEncoderSuite))
}
