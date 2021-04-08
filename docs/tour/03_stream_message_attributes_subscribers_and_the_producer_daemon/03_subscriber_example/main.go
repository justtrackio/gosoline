package main

import (
	"context"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	application.RunMdlSubscriber(transformers)
}

var transformers mdlsub.TransformerMapTypeVersionFactories = map[string]mdlsub.TransformerMapVersionFactories{
	"gosoline.stream-example.example.record": map[int]mdlsub.TransformerFactory{
		0: mdlsub.NewGenericTransformer(NewRecordTransformer()),
	},
}

type recordTransformer struct {
}

func NewRecordTransformer() *recordTransformer {
	go provideFakeData()

	return &recordTransformer{}
}

type RecordInputV0 struct {
	Id         string    `json:"id"`
	OrderDate  time.Time `json:"orderDate"`
	CustomerId uint      `json:"customerId"`
}

type Record struct {
	Id        string    `json:"id"`
	OrderDate time.Time `json:"orderDate"`
}

func (r *Record) GetId() interface{} {
	return r.Id
}

func (r recordTransformer) GetInput() interface{} {
	return &RecordInputV0{}
}

func (r recordTransformer) Transform(_ context.Context, inp interface{}) (mdlsub.Model, error) {
	input := inp.(*RecordInputV0)

	if input.CustomerId%2 == 0 {
		return nil, nil
	}

	return &Record{
		Id:        input.Id,
		OrderDate: input.OrderDate,
	}, nil
}

func provideFakeData() {
	input := stream.ProvideInMemoryInput("subscriber-record", &stream.InMemorySettings{
		Size: 3,
	})

	attributes := mdlsub.CreateMessageAttributes(mdl.ModelId{
		Project:     "gosoline",
		Family:      "stream-example",
		Application: "example",
		Name:        "record",
	}, mdlsub.TypeCreate, 0)

	// language=JSON
	msg1 := `{
		"id": "record1",
		"orderDate": "2020-02-24T12:23:00Z",
		"customerId": 15
	}`
	// language=JSON
	msg2 := `{
		"id": "record2",
		"orderDate": "2020-02-29T14:55:02Z",
		"customerId": 16
	}`
	// language=JSON
	msg3 := `{
		"id": "record3",
		"orderDate": "2020-03-12T16:07:24Z",
		"customerId": 17
	}`

	input.Publish(stream.NewJsonMessage(msg1, attributes))
	input.Publish(stream.NewJsonMessage(msg2, attributes))
	input.Publish(stream.NewJsonMessage(msg3, attributes))

	input.Stop()
}
