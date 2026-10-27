package stream_test

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"testing"
	"time"

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
