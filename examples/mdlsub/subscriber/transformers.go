package main

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/mdlsub"
)

var Transformers = mdlsub.TransformerMapTypeVersionFactories{
	"gosoline.example.mdlsub-publisher.random-number": {
		0: mdlsub.NewGenericTransformer[RandomNumber, RandomNumber](randomNumberTransformer{}),
	},
}

type RandomNumber struct {
	Id     string `json:"id"`
	Number int    `json:"number"`
}

func (n RandomNumber) GetId() any {
	return n.Id
}

type randomNumberTransformer struct{}

func (r randomNumberTransformer) Transform(_ context.Context, number RandomNumber) (out *RandomNumber, err error) {
	return &number, nil
}
