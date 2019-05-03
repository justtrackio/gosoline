package sub

import (
	"context"
	"encoding/json"
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

type ModelTransformer interface {
	GetInput() interface{}
	Transform(ctx context.Context, inp interface{}) (out Model, err error)
}

type ModelMsgTransformer func(ctx context.Context, msg *stream.ModelMsg) (out Model, err error)
type TransformerMapVersion map[int]ModelTransformer
type TransformerMapVersionFactories map[int]TransformerFactory
type TransformerMapTypeVersionFactories map[string]TransformerMapVersionFactories

type TransformerFactory func(config cfg.Config, logger mon.Logger) ModelTransformer

func BuildTransformer(modelTransformer TransformerMapVersion) ModelMsgTransformer {
	return func(ctx context.Context, msg *stream.ModelMsg) (Model, error) {
		if _, ok := modelTransformer[msg.Version]; !ok {
			return nil, fmt.Errorf("there is no transformer for modelId %s and version %d", msg.ModelId, msg.Version)
		}

		input := modelTransformer[msg.Version].GetInput()
		err := json.Unmarshal([]byte(msg.Body), input)

		if err != nil {
			return nil, errors.Wrapf(err, "can not unmarshal body for modelId %s and version %d", msg.ModelId, msg.Version)
		}

		model, err := modelTransformer[msg.Version].Transform(ctx, input)

		if err != nil {
			return nil, errors.Wrapf(err, "can not transform body for modelId %s and version %d", msg.ModelId, msg.Version)
		}

		return model, nil
	}
}
