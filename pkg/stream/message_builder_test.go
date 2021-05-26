package stream_test

import (
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/stream/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
)

//go:generate protoc --go_out=../../../../../.. message_builder_test.proto
type TestMessage struct {
	FirstField  string `json:"first_field"`
	SecondField uint32 `json:"second_field"`
}

var _ stream.ProtobufEncodable = &TestMessage{}

func (m *TestMessage) ToMessage() (proto.Message, error) {
	return &testdata.TestMessage{
		FirstField:  m.FirstField,
		SecondField: m.SecondField,
	}, nil
}

func (m *TestMessage) EmptyMessage() proto.Message {
	return &testdata.TestMessage{}
}

func (m *TestMessage) FromMessage(message proto.Message) error {
	msg := message.(*testdata.TestMessage)

	*m = TestMessage{
		FirstField:  msg.GetFirstField(),
		SecondField: msg.GetSecondField(),
	}

	return nil
}

func TestNewMessage(t *testing.T) {
	msg := stream.NewMessage(`{"foo": "bar"}`, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1": 2,
			"attribute2": "value",
		},
		Body: `{"foo": "bar"}`,
	}

	assert.Equal(t, expectedMsg, msg)
}

func TestNewJsonMessage(t *testing.T) {
	msg := stream.NewJsonMessage(`{"foo": "bar"}`, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1":             2,
			"attribute2":             "value",
			stream.AttributeEncoding: stream.EncodingJson,
		},
		Body: `{"foo": "bar"}`,
	}

	assert.Equal(t, expectedMsg, msg)
}

func TestNewProtobufMessage(t *testing.T) {
	msg := stream.NewProtobufMessage(string([]byte{10, 3, 102, 111, 111, 16, 42}), map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1":             2,
			"attribute2":             "value",
			stream.AttributeEncoding: stream.EncodingProtobuf,
		},
		Body: string([]byte{10, 3, 102, 111, 111, 16, 42}),
	}

	assert.Equal(t, expectedMsg, msg)
}

func TestMarshalJsonMessage(t *testing.T) {
	msg, err := stream.MarshalJsonMessage(&TestMessage{
		FirstField:  "foo",
		SecondField: 42,
	}, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1":             2,
			"attribute2":             "value",
			stream.AttributeEncoding: stream.EncodingJson,
		},
		Body: `{"first_field":"foo","second_field":42}`,
	}

	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}

func TestMarshalProtobufMessage(t *testing.T) {
	msg, err := stream.MarshalProtobufMessage(&TestMessage{
		FirstField:  "foo",
		SecondField: 42,
	}, map[string]interface{}{
		"attribute1": 2,
		"attribute2": "value",
	})

	expectedMsg := &stream.Message{
		Attributes: map[string]interface{}{
			"attribute1":             2,
			"attribute2":             "value",
			stream.AttributeEncoding: stream.EncodingProtobuf,
		},
		Body: string([]byte{10, 3, 102, 111, 111, 16, 42}),
	}

	assert.NoError(t, err)
	assert.Equal(t, expectedMsg, msg)
}
