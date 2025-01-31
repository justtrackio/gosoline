package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

var Transformers = mdlsub.TransformerMapTypeVersionFactories{
	"gosoline.example.mdlsub-publisher.random-number": {
		0: mdlsub.NewGenericTransformer(&randomNumberTransformer{}),
	},
}

type RandomNumber struct {
	Id     string `json:"id"`
	Number int    `json:"number"`
}

func (n RandomNumber) GetId() interface{} {
	return n.Id
}

type randomNumberTransformer struct{}

func (r randomNumberTransformer) GetInput() interface{} {
	return &RandomNumber{}
}

func (r randomNumberTransformer) GetModel() any {
	return &RandomNumber{}
}

func (r randomNumberTransformer) Transform(_ context.Context, inp interface{}) (out mdlsub.Model, err error) {
	number := inp.(*RandomNumber)

	return number, nil
}
