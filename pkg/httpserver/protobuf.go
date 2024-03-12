package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin/binding"
	"google.golang.org/protobuf/proto"
)

// ProtobufDecodable allows CreateProtobufHandler to decode the request body via protobuf.
type ProtobufDecodable interface {
	// EmptyMessage should provide an empty instance of the message format to decode from.
	EmptyMessage() proto.Message
	// FromMessage should extract the data from the message and write it to the receiver.
	FromMessage(message proto.Message) error
}

// ProtobufEncodable is required for NewProtobufResponse to create the protobuf response.
type ProtobufEncodable interface {
	// ToMessage turns the response value into a protobuf message representation
	ToMessage() (proto.Message, error)
}

type protobufMessageBinding struct{}

var protobufBinding = protobufMessageBinding{}

func (p protobufMessageBinding) Name() string {
	return binding.ProtoBuf.Name()
}

func (p protobufMessageBinding) Bind(request *http.Request, bodyI any) error {
	body, ok := bodyI.(ProtobufDecodable)
	if !ok {
		return fmt.Errorf("body was not protobuf encodable: %T", bodyI)
	}

	message := body.EmptyMessage()
	if err := binding.ProtoBuf.Bind(request, message); err != nil {
		return err
	}

	if err := body.FromMessage(message); err != nil {
		return err
	}

	if binding.Validator == nil {
		return nil
	}

	return binding.Validator.ValidateStruct(body)
}
