package retry_handler_sqs

import (
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/test/stream/retry_handler_sqs/testdata"
	"google.golang.org/protobuf/proto"
)

//go:generate protoc --go_out=.. data_model.proto
type DataModel struct {
	Id int64
}

var _ stream.ProtobufEncodable = &DataModel{}

func (m *DataModel) ToMessage() (proto.Message, error) {
	return &testdata.DataModel{
		Id: m.Id,
	}, nil
}

func (m *DataModel) EmptyMessage() proto.Message {
	return &testdata.DataModel{}
}

func (m *DataModel) FromMessage(message proto.Message) error {
	msg := message.(*testdata.DataModel)

	*m = DataModel{
		Id: msg.GetId(),
	}

	return nil
}
