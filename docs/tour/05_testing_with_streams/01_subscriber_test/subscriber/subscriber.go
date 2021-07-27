package subscriber

import (
	"context"
	"github.com/applike/gosoline/pkg/mdlsub"
)

type BookInput struct {
	Name    string `json:"name"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

type Book struct {
	Name   string `json:"name" ddb:"key=hash"`
	Author string `json:"author" ddb:"key=range"`
	Length int    `json:"length"`
}

func (b Book) GetId() interface{} {
	return b.Name
}

type bookTransformerV0 struct{}

func (b bookTransformerV0) GetInput() interface{} {
	return &BookInput{}
}

func (b bookTransformerV0) Transform(ctx context.Context, inp interface{}) (out mdlsub.Model, err error) {
	input := inp.(*BookInput)

	return &Book{
		Name:   input.Name,
		Author: input.Author,
		Length: len(input.Content),
	}, nil
}

var Transformers = mdlsub.TransformerMapTypeVersionFactories{
	"gosoline.subscriber-test.book-store.book": {
		0: mdlsub.NewGenericTransformer(bookTransformerV0{}),
	},
}
