package sub

import (
	"context"
	"encoding/json"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
)

type subOutBlob struct {
	logger mon.Logger
	store  blob.Store
}

func (p *subOutBlob) Boot(config cfg.Config, logger mon.Logger, settings Settings) error {
	p.logger = logger
	p.store = blob.NewStore(config, logger, blob.Settings{
		Prefix: settings.TargetModelId.String(),
	})

	return nil
}

func (p *subOutBlob) Persist(ctx context.Context, model Model, op string) error {
	id := model.GetId()
	idString := ""

	switch id.(type) {
	case string:
		idString = id.(string)
	}

	bytes, err := json.Marshal(model)

	if err != nil {
		return err
	}

	obj := &blob.Object{
		Key:  mdl.String(idString),
		Body: blob.StreamBytes(bytes),
	}

	err = p.store.WriteOne(obj)

	return err
}
