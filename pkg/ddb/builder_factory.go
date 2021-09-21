package ddb

import "github.com/justtrackio/gosoline/pkg/clock"

type BuilderFactory interface {
	GetItemBuilder() GetItemBuilder
	QueryBuilder() QueryBuilder
	BatchGetItemsBuilder() BatchGetItemsBuilder
	PutItemBuilder() PutItemBuilder
	UpdateItemBuilder() UpdateItemBuilder
}

type builderFactory struct {
	metadata *Metadata
	clock    clock.Clock
}

func NewBuilderFactory(settings *Settings, clock clock.Clock) (BuilderFactory, error) {
	metadataFactory := NewMetadataFactory()
	metadata, err := metadataFactory.GetMetadata(settings)
	if err != nil {
		return nil, err
	}

	return &builderFactory{
		metadata: metadata,
		clock:    clock,
	}, nil
}

func (f *builderFactory) GetItemBuilder() GetItemBuilder {
	return NewGetItemBuilder(f.metadata, f.clock)
}

func (f *builderFactory) QueryBuilder() QueryBuilder {
	return NewQueryBuilder(f.metadata, f.clock)
}

func (f *builderFactory) BatchGetItemsBuilder() BatchGetItemsBuilder {
	return NewBatchGetItemsBuilder(f.metadata, f.clock)
}

func (f *builderFactory) PutItemBuilder() PutItemBuilder {
	return NewPutItemBuilder(f.metadata)
}

func (f *builderFactory) UpdateItemBuilder() UpdateItemBuilder {
	return NewUpdateItemBuilder(f.metadata)
}
