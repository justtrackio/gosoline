package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
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

//go:generate mockery --name ModelTransformer
type ModelTransformer interface {
	GetInput() any
	Transform(ctx context.Context, inp any) (out Model, err error)
}

type (
	ModelTransformers                  map[string]VersionedModelTransformers
	TransformerFactory                 func(ctx context.Context, config cfg.Config, logger log.Logger) (ModelTransformer, error)
	TransformerMapTypeVersionFactories map[string]TransformerMapVersionFactories
	TransformerMapVersionFactories     map[int]TransformerFactory
	VersionedModelTransformers         map[int]ModelTransformer
)

func initTransformers(ctx context.Context, config cfg.Config, logger log.Logger, subscriberSettings map[string]*SubscriberSettings, transformerFactories TransformerMapTypeVersionFactories) (ModelTransformers, error) {
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

func NewGenericTransformer(transformer ModelTransformer) func(context.Context, cfg.Config, log.Logger) (ModelTransformer, error) {
	return func(_ context.Context, _ cfg.Config, _ log.Logger) (ModelTransformer, error) {
		return transformer, nil
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
