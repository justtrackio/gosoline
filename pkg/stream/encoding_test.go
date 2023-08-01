package stream_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/testdata"
	"github.com/stretchr/testify/assert"
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
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"id":42,"data":"this is data!"}`), body)

	out := &TestEncodingMessage{}
	err = stream.DecodeMessage(stream.EncodingJson, body, nil, out)
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
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("CCoSDXRoaXMgaXMgZGF0YSE="), body)

	out := &TestEncodingMessage{}
	err = stream.DecodeMessage(stream.EncodingProtobuf, body, nil, out)
	assert.NoError(t, err)
	assert.Equal(t, &TestEncodingMessage{
		Id:   42,
		Data: "this is data!",
	}, out)
}
