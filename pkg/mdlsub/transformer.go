package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/spf13/cast"
)

type Model interface {
	GetId() interface{}
}

type ModelDb struct {
	Id *uint `gorm:"primary_key;"`
}

func (m ModelDb) GetId() interface{} {
	return *m.Id
}

//go:generate mockery -name ModelTransformer
type ModelTransformer interface {
	GetInput() interface{}
	Transform(ctx context.Context, inp interface{}) (out Model, err error)
}

type ModelTransformers map[string]VersionedModelTransformers
type VersionedModelTransformers map[int]ModelTransformer

type TransformerFactory func(config cfg.Config, logger log.Logger) (ModelTransformer, error)
type TransformerMapVersionFactories map[int]TransformerFactory
type TransformerMapTypeVersionFactories map[string]TransformerMapVersionFactories

func initTransformers(config cfg.Config, logger log.Logger, subscriberSettings map[string]*SubscriberSettings, transformerFactories TransformerMapTypeVersionFactories) (ModelTransformers, error) {
	var err error
	var transformers = make(ModelTransformers)

	for name, settings := range subscriberSettings {
		modelId := settings.SourceModel.String()

		if _, ok := transformerFactories[modelId]; !ok {
			return nil, fmt.Errorf("there is no transformer for subscriber %s with modelId %s", name, modelId)
		}
	}

	for modelId, versionedFactories := range transformerFactories {
		transformers[modelId] = make(map[int]ModelTransformer)

		for version, factory := range versionedFactories {
			if transformers[modelId][version], err = factory(config, logger); err != nil {
				return nil, fmt.Errorf("can not create transformer for modelId %s in version %d: %w", modelId, version, err)
			}
		}
	}

	return transformers, nil
}

func NewGenericTransformer(transformer ModelTransformer) func(cfg.Config, log.Logger) (ModelTransformer, error) {
	return func(_ cfg.Config, _ log.Logger) (ModelTransformer, error) {
		return transformer, nil
	}
}

type ModelSpecification struct {
	CrudType string
	Version  int
	ModelId  string
}

func getModelSpecification(attributes map[string]interface{}) (*ModelSpecification, error) {
	var ok bool
	var err error
	var spec ModelSpecification

	if _, ok = attributes["type"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'type'")
	}

	if spec.CrudType, ok = attributes["type"].(string); !ok {
		return nil, fmt.Errorf("type is not a string: %v", attributes["type"])
	}

	if _, ok := attributes["version"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'version'")
	}

	if spec.Version, err = cast.ToIntE(attributes["version"]); err != nil {
		return nil, fmt.Errorf("version is not an int: %v", attributes["version"])
	}

	if _, ok = attributes["modelId"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'modelId'")
	}

	if spec.ModelId, ok = attributes["modelId"].(string); !ok {
		return nil, fmt.Errorf("modelId is not a string: %v", attributes["modelId"])
	}

	return &spec, nil
}
