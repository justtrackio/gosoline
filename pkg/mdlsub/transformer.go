package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/spf13/cast"
)

type Model interface {
	GetId() any
}

type ModelDb struct {
	Id *uint `gorm:"primary_key;"`
}

func (m ModelDb) GetId() any {
	return *m.Id
}

// A ModelTransformer performs the actual transformation work, but should not be directly be implemented by a developer. Instead, implement a
// TypedTransformer and convert it to a ModelTransformer using NewGenericTransformer, EraseTransformerFactoryTypes, or EraseTransformerTypes.
type ModelTransformer interface {
	getInput() any
	GetModel() (any, error)
	transform(ctx context.Context, inp any) (out Model, err error)
	getSchemaSettings() (*stream.SchemaSettings, error)
}

// A TypedTransformer implements a subscriber. For every item it has to transform it into the corresponding persisted model. If it returns nil, the
// item will not be persisted. However, if the item has been persisted before and nil is returned, the item will not be updated or deleted.
//
//go:generate go run github.com/vektra/mockery/v2 --name TypedTransformer
type TypedTransformer[I any, M Model] interface {
	// Transform converts the input into an output model. If Transform returns nil as the output, we don't persist the value.
	Transform(ctx context.Context, inp I) (out *M, err error)
}

type untypedTransformer[I any, M Model] struct {
	transformer TypedTransformer[I, M]
}

func (u untypedTransformer[I, M]) getInput() any {
	return new(I)
}

func (u untypedTransformer[I, M]) GetModel() (any, error) {
	return new(M), nil
}

func (u untypedTransformer[I, M]) getSchemaSettings() (*stream.SchemaSettings, error) {
	if schemaAware, ok := u.transformer.(stream.SchemaSettingsAwareCallback); ok {
		return schemaAware.GetSchemaSettings()
	}

	return nil, nil
}

func (u untypedTransformer[I, M]) transform(ctx context.Context, inp any) (out Model, err error) {
	input := inp.(*I)
	output, err := u.transformer.Transform(ctx, *input)
	if err != nil || output == nil {
		return nil, err
	}

	return *output, err
}

type (
	ModelTransformers                       map[string]VersionedModelTransformers
	TransformerFactory                      func(ctx context.Context, config cfg.Config, logger log.Logger) (ModelTransformer, error)
	TypedTransformerFactory[I any, M Model] func(ctx context.Context, config cfg.Config, logger log.Logger) (TypedTransformer[I, M], error)
	TransformerMapTypeVersionFactories      map[string]TransformerMapVersionFactories
	TransformerMapVersionFactories          map[int]TransformerFactory
	VersionedModelTransformers              map[int]ModelTransformer
)

func initTransformers(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	subscriberSettings map[string]*SubscriberSettings,
	transformerFactories TransformerMapTypeVersionFactories,
) (ModelTransformers, error) {
	var err error
	transformers := make(ModelTransformers)

	for name, settings := range subscriberSettings {
		modelId := settings.SourceModel.String()

		if _, ok := transformerFactories[modelId]; !ok {
			return nil, fmt.Errorf("can not create transformers: there is no transformer for subscriber %s with modelId %s", name, modelId)
		}
	}

	for modelId, versionedFactories := range transformerFactories {
		transformers[modelId] = make(map[int]ModelTransformer)

		for version, factory := range versionedFactories {
			if transformers[modelId][version], err = factory(ctx, config, logger); err != nil {
				return nil, fmt.Errorf("can not create transformer for modelId %s in version %d: %w", modelId, version, err)
			}
		}
	}

	return transformers, nil
}

// NewGenericTransformer removes the types from a TypedTransformer and turns a transformer value into a TransformerFactory of that value.
func NewGenericTransformer[I any, M Model](transformer TypedTransformer[I, M]) func(context.Context, cfg.Config, log.Logger) (ModelTransformer, error) {
	return func(_ context.Context, _ cfg.Config, _ log.Logger) (ModelTransformer, error) {
		return EraseTransformerTypes(transformer), nil
	}
}

// EraseTransformerFactoryTypes takes a TypedTransformerFactory and turns it into an untyped transformer factory, allowing you to embed it into a list
// of transformers.
func EraseTransformerFactoryTypes[I any, M Model](transformerFactory TypedTransformerFactory[I, M]) TransformerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (ModelTransformer, error) {
		transformer, err := transformerFactory(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return EraseTransformerTypes(transformer), nil
	}
}

// EraseTransformerTypes takes a TypedTransformer and turns it into an untyped ModelTransformer.
func EraseTransformerTypes[I any, M Model](transformer TypedTransformer[I, M]) ModelTransformer {
	return untypedTransformer[I, M]{
		transformer: transformer,
	}
}

type ModelSpecification struct {
	CrudType string
	Version  int
	ModelId  string
}

func (m ModelSpecification) String() string {
	return fmt.Sprintf("[%s]%s@v%d", m.CrudType, m.ModelId, m.Version)
}

func getModelSpecification(attributes map[string]string) (*ModelSpecification, error) {
	var ok bool
	var err error
	var spec ModelSpecification

	if _, ok = attributes["type"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'type'")
	}

	spec.CrudType = attributes["type"]

	if _, ok := attributes["version"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'version'")
	}

	if spec.Version, err = cast.ToIntE(attributes["version"]); err != nil {
		return nil, fmt.Errorf("version is not an int: %v", attributes["version"])
	}

	if _, ok = attributes["modelId"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'modelId'")
	}

	spec.ModelId = attributes["modelId"]

	return &spec, nil
}
