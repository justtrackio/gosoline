package stream_test

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/iskorotkov/avro/v2"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=.. encoding_test.proto
type TestEncodingMessage struct {
	Id   int    `json:"id"`
	Data string `json:"data"`
}

var _ stream.ProtobufEncodable = &TestEncodingMessage{}

func (m *TestEncodingMessage) ToMessage() (proto.Message, error) {
	return &testdata.TestEncodingMessage{
		Id:   int32(m.Id),
		Data: m.Data,
	}, nil
}

func (m *TestEncodingMessage) EmptyMessage() proto.Message {
	return &testdata.TestEncodingMessage{}
}

func (m *TestEncodingMessage) FromMessage(message proto.Message) error {
	msg := message.(*testdata.TestEncodingMessage)

	*m = TestEncodingMessage{
		Id:   int(msg.GetId()),
		Data: msg.GetData(),
	}

	return nil
}

func TestEncodingJson(t *testing.T) {
	body, err := stream.EncodeMessage(stream.EncodingJson, &TestEncodingMessage{
		Id:   42,
		Data: "this is data!",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"id":42,"data":"this is data!"}`), body)

	out := &TestEncodingMessage{}
	err = stream.DecodeMessage(stream.EncodingJson, body, out)
	assert.NoError(t, err)
	assert.Equal(t, &TestEncodingMessage{
		Id:   42,
		Data: "this is data!",
	}, out)
}

func TestEncodingProtobuf(t *testing.T) {
	body, err := stream.EncodeMessage(stream.EncodingProtobuf, &TestEncodingMessage{
		Id:   42,
		Data: "this is data!",
	})
	assert.NoError(t, err)
	assert.Equal(t, []byte("CCoSDXRoaXMgaXMgZGF0YSE="), body)

	out := &TestEncodingMessage{}
	err = stream.DecodeMessage(stream.EncodingProtobuf, body, out)
	assert.NoError(t, err)
	assert.Equal(t, &TestEncodingMessage{
		Id:   42,
		Data: "this is data!",
	}, out)
}

func TestEncodingAvroHistoricalRoundTrip(t *testing.T) {
	type message struct {
		Id         int               `avro:"id"`
		Data       string            `avro:"data"`
		Attributes map[string]string `avro:"attributes"`
		Values     []string          `avro:"values"`
	}

	encoder, err := stream.NewAvroEncoder(`{
		"type": "record",
		"name": "HistoricalMessage",
		"fields": [
			{"name": "id", "type": "int"},
			{"name": "data", "type": "string"},
			{"name": "attributes", "type": {"type": "map", "values": "string"}},
			{"name": "values", "type": {"type": "array", "items": "string"}}
		]
	}`)
	require.NoError(t, err)

	// Pin the historical wire format, including the default size-prefixed blocks.
	body := []byte{
		0x54,                // id: 42
		0x06, 'o', 'l', 'd', // data: "old"
		0x01, 0x10, // map: one entry, eight encoded bytes
		0x06, 'f', 'o', 'o', // key: "foo"
		0x06, 'b', 'a', 'r', 0x00, // value: "bar", end map
		0x03, 0x10, // array: two entries, eight encoded bytes
		0x06, 'o', 'n', 'e',
		0x06, 't', 'w', 'o', 0x00,
	}
	expected := message{
		Id:         42,
		Data:       "old",
		Attributes: map[string]string{"foo": "bar"},
		Values:     []string{"one", "two"},
	}

	var decoded message
	require.NoError(t, encoder.Decode(body, &decoded))
	assert.Equal(t, expected, decoded)

	encoded, err := encoder.Encode(expected)
	require.NoError(t, err)
	assert.Equal(t, body, encoded)
}

func TestEncodingAvroWithCodecValidation(t *testing.T) {
	encoder, err := stream.NewAvroEncoderWithCodec(`"int"`, nil)
	require.ErrorContains(t, err, "avro codec must not be nil")
	assert.Nil(t, encoder)

	for _, newEncoder := range []func(string) (stream.MessageBodyEncoder, error){
		stream.NewAvroEncoder,
		func(schema string) (stream.MessageBodyEncoder, error) {
			return stream.NewAvroEncoderWithCodec(schema, avro.Config{}.Freeze())
		},
	} {
		encoder, err = newEncoder(`{"type":`)
		require.ErrorContains(t, err, "failed to parse avro schema")
		assert.Nil(t, encoder)
	}
}

func TestEncodingAvroWithCodecMapLimit(t *testing.T) {
	const schema = `{"type":"map","values":"int"}`
	limited, err := stream.NewAvroEncoderWithCodec(schema, avro.Config{MaxMapAllocSize: 2}.Freeze())
	require.NoError(t, err)
	independent, err := stream.NewAvroEncoderWithCodec(schema, avro.Config{}.Freeze())
	require.NoError(t, err)
	defaultEncoder, err := stream.NewAvroEncoder(schema)
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		body []byte
		size int
	}{
		{"single_block_boundary", []byte{4, 2, 'a', 2, 2, 'b', 4, 0}, 2},
		{"multiple_blocks_boundary", []byte{2, 2, 'a', 2, 2, 2, 'b', 4, 0}, 2},
		{"sized_blocks_boundary", []byte{1, 6, 2, 'a', 2, 1, 6, 2, 'b', 4, 0}, 2},
		{"multiple_blocks_exceeded", []byte{2, 2, 'a', 2, 2, 2, 'b', 4, 2, 2, 'c', 6, 0}, 3},
		{"sized_blocks_exceeded", []byte{1, 6, 2, 'a', 2, 1, 6, 2, 'b', 4, 1, 6, 2, 'c', 6, 0}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Exercise both typed-map and generic-map decoding.
			var typed map[string]int
			var generic any
			for _, out := range []any{&typed, &generic} {
				if tc.size > 2 {
					require.ErrorContains(t, limited.Decode(tc.body, out), "Config.MaxMapAllocSize")
				} else {
					require.NoError(t, limited.Decode(tc.body, out))
				}
			}
			for _, encoder := range []stream.MessageBodyEncoder{independent, defaultEncoder} {
				var decoded map[string]int
				require.NoError(t, encoder.Decode(tc.body, &decoded))
				assert.Len(t, decoded, tc.size)
			}
		})
	}
}

func TestEncodingAvroWithCodecByteLimit(t *testing.T) {
	encoder, err := stream.NewAvroEncoderWithCodec(`"string"`, avro.Config{MaxByteSliceSize: 3}.Freeze())
	require.NoError(t, err)

	for _, value := range []string{"abc", "abcd"} {
		// The caller's allocation limit applies to decoding, not encoding.
		body, err := encoder.Encode(value)
		require.NoError(t, err)
		var decoded string
		if len(value) > 3 {
			require.ErrorContains(t, encoder.Decode(body, &decoded), "Config.MaxByteSliceSize")
		} else {
			require.NoError(t, encoder.Decode(body, &decoded))
			assert.Equal(t, value, decoded)
		}
	}
}

func TestEncodingAvroWithCodecRegistrations(t *testing.T) {
	type firstEvent struct {
		Value int `avro:"value"`
	}
	type secondEvent struct {
		Value int `avro:"value"`
	}
	const schema = `["null",{"type":"record","name":"CallerCodecEvent","fields":[{"name":"value","type":"int"}]}]`
	body := []byte{2, 84}
	for _, event := range []any{firstEvent{Value: 42}, secondEvent{Value: 42}} {
		codec := avro.Config{UnionResolutionError: true}.Freeze()
		codec.Register("CallerCodecEvent", event)
		encoder, err := stream.NewAvroEncoderWithCodec(schema, codec)
		require.NoError(t, err)
		encoded, err := encoder.Encode(event)
		require.NoError(t, err)
		assert.Equal(t, body, encoded)
		var decoded any
		require.NoError(t, encoder.Decode(body, &decoded))
		assert.Equal(t, event, decoded)
	}

	unregistered, err := stream.NewAvroEncoderWithCodec(schema, avro.Config{UnionResolutionError: true}.Freeze())
	require.NoError(t, err)
	_, err = unregistered.Encode(firstEvent{Value: 42})
	require.Error(t, err)
	var decoded any
	require.Error(t, unregistered.Decode(body, &decoded))
	defaultEncoder, err := stream.NewAvroEncoder(schema)
	require.NoError(t, err)
	require.NoError(t, defaultEncoder.Decode(body, &decoded))
	assert.Equal(t, map[string]any{"CallerCodecEvent": map[string]any{"value": 42}}, decoded)
}

func TestEncodingAvroWithCodecConverters(t *testing.T) {
	const schema = `{"type":"record","name":"ConvertedEvent","fields":[{"name":"value","type":["null","boolean"]}]}`
	codec := avro.Config{}.Freeze()
	codec.RegisterTypeConverters(avro.TypeConversionFuncs{
		AvroType: avro.Union,
		EncoderTypeConversion: func(in any, _ avro.Schema) (any, error) {
			return in == "yes", nil
		},
		DecoderTypeConversion: func(in any, _ avro.Schema) (any, error) {
			if in == true {
				return "yes", nil
			}

			return "no", nil
		},
	})
	encoder, err := stream.NewAvroEncoderWithCodec(schema, codec)
	require.NoError(t, err)
	body, err := encoder.Encode(map[string]any{"value": "yes"})
	require.NoError(t, err)
	assert.Equal(t, []byte{2, 1}, body)
	var decoded map[string]any
	require.NoError(t, encoder.Decode(body, &decoded))
	assert.Equal(t, map[string]any{"value": "yes"}, decoded)

	independent, err := stream.NewAvroEncoderWithCodec(schema, avro.Config{}.Freeze())
	require.NoError(t, err)
	defaultEncoder, err := stream.NewAvroEncoder(schema)
	require.NoError(t, err)
	for _, other := range []stream.MessageBodyEncoder{independent, defaultEncoder} {
		_, err = other.Encode(map[string]any{"value": "yes"})
		require.Error(t, err)
		require.NoError(t, other.Decode(body, &decoded))
		assert.Equal(t, map[string]any{"value": true}, decoded)
	}
}

func TestEncodingAvroDefaultCodecResolution(t *testing.T) {
	const helperEnv = "GOSOLINE_AVRO_DEFAULT_CODEC_HELPER"
	if os.Getenv(helperEnv) != "1" {
		// Replacing DefaultConfig must not race with unrelated stream users.
		executable, err := os.Executable()
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestEncodingAvroDefaultCodecResolution$", "-test.count=1")
		cmd.Env = append(os.Environ(), helperEnv+"=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, ctx.Err(), "default codec test timed out: %s", output)
		require.NoError(t, err, "default codec test failed: %s", output)

		return
	}

	type event struct {
		Value int `avro:"value"`
	}
	const schema = `["null",{"type":"record","name":"GlobalCodecEvent","fields":[{"name":"value","type":"int"}]}]`
	encoder, err := stream.NewAvroEncoder(schema)
	require.NoError(t, err)
	// Replace the default after construction and register on that replacement.
	avro.DefaultConfig = avro.Config{UnionResolutionError: true}.Freeze()
	avro.Register("GlobalCodecEvent", event{})
	body, err := encoder.Encode(event{Value: 42})
	require.NoError(t, err)
	assert.Equal(t, []byte{2, 84}, body)
	var decoded any
	require.NoError(t, encoder.Decode(body, &decoded))
	assert.Equal(t, event{Value: 42}, decoded)

	// The same encoder must resolve the default again for every operation.
	avro.DefaultConfig = avro.Config{UnionResolutionError: true}.Freeze()
	_, err = encoder.Encode(event{Value: 42})
	require.Error(t, err)
	require.Error(t, encoder.Decode(body, &decoded))
}

func TestEncodingAvroMalformedSkippedBlocks(t *testing.T) {
	const helperEnv = "GOSOLINE_AVRO_SKIPPED_BLOCK_HELPER"
	if os.Getenv(helperEnv) == "1" {
		decodeMalformedAvroSkippedBlocks(t)

		return
	}

	// A decoder regression must not strand a goroutine or hang the test process.
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestEncodingAvroMalformedSkippedBlocks$", "-test.count=1")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "malformed skipped Avro block decoding timed out: %s", output)
	require.NoError(t, err, "malformed skipped Avro block decoding failed: %s", output)
}

func decodeMalformedAvroSkippedBlocks(t *testing.T) {
	t.Helper()

	for _, collection := range []struct {
		name    string
		schema  string
		payload []byte
	}{
		{name: "array", schema: `{"type":"array","items":"int"}`, payload: []byte{0x36, 0x36}},
		{name: "map", schema: `{"type":"map","values":"int"}`, payload: []byte{0x02, 'f'}},
	} {
		t.Run(collection.name, func(t *testing.T) {
			encoder, err := stream.NewAvroEncoder(`{
				"type":"record",
				"name":"SkippedCollection",
				"fields":[
					{"name":"ignored","type":` + collection.schema + `},
					{"name":"keep","type":"int"}
				]
			}`)
			require.NoError(t, err)

			for _, tc := range []struct {
				name    string
				body    []byte
				wantErr string
			}{
				{
					name:    "huge_count_truncated",
					body:    append(binary.AppendVarint(nil, math.MaxInt32), collection.payload...),
					wantErr: "unexpected EOF",
				},
				{
					name:    "huge_count_zero_size_truncated",
					body:    append(binary.AppendVarint(binary.AppendVarint(nil, -math.MaxInt32), 0), collection.payload...),
					wantErr: "unexpected EOF",
				},
				{
					name:    "minimum_count_overflow",
					body:    binary.AppendVarint(nil, math.MinInt64),
					wantErr: "block length is too small",
				},
				{
					name:    "negative_block_size",
					body:    binary.AppendVarint(binary.AppendVarint(nil, -1), -1),
					wantErr: "skip size is too small",
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					// Omit the collection from the destination so even the huge
					// count fixtures cannot allocate a materialized map or array.
					var decoded struct {
						Keep int `avro:"keep"`
					}
					require.ErrorContains(t, encoder.Decode(tc.body, &decoded), tc.wantErr)
				})
			}
		})
	}
}
