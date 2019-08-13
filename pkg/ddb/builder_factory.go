package ddb

type BuilderFactory interface {
	GetItemBuilder() GetItemBuilder
	QueryBuilder() QueryBuilder
	BatchGetItemsBuilder() BatchGetItemsBuilder
	PutItemBuilder() PutItemBuilder
	UpdateItemBuilder() UpdateItemBuilder
}

type builderFactory struct {
	metadata *Metadata
}

func NewBuilderFactory(settings *Settings) (BuilderFactory, error) {
	metadataFactory := NewMetadataFactory()
	metadata, err := metadataFactory.GetMetadata(settings)

	if err != nil {
		return nil, err
	}

	return &builderFactory{
		metadata: metadata,
	}, nil
}

func (f *builderFactory) GetItemBuilder() GetItemBuilder {
	return NewGetItemBuilder(f.metadata)
}

func (f *builderFactory) QueryBuilder() QueryBuilder {
	return NewQueryBuilder(f.metadata)
}

func (f *builderFactory) BatchGetItemsBuilder() BatchGetItemsBuilder {
	return NewBatchGetItemsBuilder(f.metadata)
}

func (f *builderFactory) PutItemBuilder() PutItemBuilder {
	return NewPutItemBuilder(f.metadata)
}

func (f *builderFactory) UpdateItemBuilder() UpdateItemBuilder {
	return NewUpdateItemBuilder(f.metadata)
}
