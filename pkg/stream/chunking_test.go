package stream_test

import (
	"encoding/json"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildChunks_Single_1(t *testing.T) {
	msg := stream.NewMessage("bla", map[string]interface{}{
		"foo": "bar",
		"num": 1,
	})

	bytes, _ := json.Marshal(msg)
	batch := []stream.WritableMessage{msg}

	chunks, err := stream.BuildChunks(batch, 1)

	assert.Nil(t, err, "there should be no error")
	assert.Len(t, chunks, 1, "there should be 1 chunk")
	assert.Len(t, chunks[0], 1, "there the chunk should have a length of 1")
	assert.Equal(t, []byte(`{"attributes":{"foo":"bar","num":1},"body":"bla"}`), bytes, "the bytes should match")
}

func TestBuildChunks_Single_2(t *testing.T) {
	batch := []stream.WritableMessage{stream.NewMessage("bla")}

	chunks, err := stream.BuildChunks(batch, 500)

	assert.Nil(t, err, "there should be no error")
	assert.Len(t, chunks, 1, "there should be 1 chunk")
	assert.Len(t, chunks[0], 1, "the chunk should have a length of 1")
}

func TestBuildChunks_Batch(t *testing.T) {
	batch := []stream.WritableMessage{
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
		stream.NewMessage("bla"),
	}

	chunks, err := stream.BuildChunks(batch, 2)

	assert.Nil(t, err, "there should be no error")
	assert.Len(t, chunks, 5, "there should be 5 chunks")
	assert.Len(t, chunks[0], 2, "there the chunk should have a length of 2")
}

func TestByteChunkToInterfaces(t *testing.T) {
	batch := []stream.WritableMessage{stream.NewMessage("bla")}
	chunks, _ := stream.BuildChunks(batch, 500)

	interfaces := stream.ByteChunkToInterfaces(chunks[0])

	assert.IsType(t, []interface{}{}, interfaces)

	bytes, ok := interfaces[0].([]byte)
	assert.True(t, ok, "it should be a byte slice")
	assert.Equal(t, []byte(`{"attributes":{},"body":"bla"}`), bytes, "the bytes should match")
}

func TestByteChunkToStrings(t *testing.T) {
	i := []string{
		"test1",
		"test2",
		"test3",
	}

	chunk := stream.Chunk{
		[]byte(i[0]),
		[]byte(i[1]),
		[]byte(i[2]),
	}

	s := stream.ByteChunkToStrings(chunk)

	assert.Equal(t, i, s)
}
