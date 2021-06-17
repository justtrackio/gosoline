package stream_test

import (
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/stream/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
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
