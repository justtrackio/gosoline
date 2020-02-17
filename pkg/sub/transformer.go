package sub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/pkg/errors"
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

type ModelMsgTransformer func(ctx context.Context, spec *ModelSpecification, msg *stream.Message) (context.Context, Model, error)
type TransformerMapVersion map[int]ModelTransformer
type TransformerMapVersionFactories map[int]TransformerFactory
type TransformerMapTypeVersionFactories map[string]TransformerMapVersionFactories

type TransformerFactory func(config cfg.Config, logger mon.Logger) ModelTransformer

func BuildTransformer(modelTransformer TransformerMapVersion) ModelMsgTransformer {
	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})

	return func(ctx context.Context, spec *ModelSpecification, msg *stream.Message) (context.Context, Model, error) {
		if _, ok := modelTransformer[spec.Version]; !ok {
			return ctx, nil, fmt.Errorf("there is no transformer for modelId %s and version %d", spec.ModelId, spec.Version)
		}

		input := modelTransformer[spec.Version].GetInput()
		ctx, _, err := encoder.Decode(ctx, msg, input)

		if err != nil {
			return ctx, nil, errors.Wrapf(err, "can not decode msg for modelId %s and version %d", spec.ModelId, spec.Version)
		}

		model, err := modelTransformer[spec.Version].Transform(ctx, input)

		if err != nil {
			return ctx, nil, errors.Wrapf(err, "can not transform body for modelId %s and version %d", spec.ModelId, spec.Version)
		}

		return ctx, model, nil
	}
}

type ModelSpecification struct {
	CrudType string
	Version  int
	ModelId  string
}

func getModelSpecification(msg *stream.Message) (*ModelSpecification, error) {
	if _, ok := msg.Attributes["type"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'type'")
	}

	crudType, ok := msg.Attributes["type"].(string)

	if !ok {
		return nil, fmt.Errorf("type is not a string: %v", msg.Attributes["type"])
	}

	if _, ok := msg.Attributes["version"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'version'")
	}

	versionFloat, ok := msg.Attributes["version"].(float64)

	if !ok {
		return nil, fmt.Errorf("version is not an int: %v", msg.Attributes["version"])
	}

	version := int(versionFloat)

	if _, ok := msg.Attributes["modelId"]; !ok {
		return nil, fmt.Errorf("the message has no attribute named 'modelId'")
	}

	modelId, ok := msg.Attributes["modelId"].(string)

	if !ok {
		return nil, fmt.Errorf("modelId is not a string: %v", msg.Attributes["modelId"])
	}

	return &ModelSpecification{
		CrudType: crudType,
		Version:  version,
		ModelId:  modelId,
	}, nil
}
