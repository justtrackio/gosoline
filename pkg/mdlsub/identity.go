package mdlsub

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

// DeriveIdentity creates a cfg.AppIdentity from a mdl.ModelId.
//
// It first pads the ModelId from config (filling missing env/app/tags from global app.*),
// then converts to AppIdentity. This provides a consistent way to derive AWS/stream
// resource identity from model identity without requiring explicit identity configuration.
//
// Precedence: explicit ModelId fields win over config-derived values.
func DeriveIdentity(config cfg.Config, modelId mdl.ModelId) (cfg.AppIdentity, error) {
	// Pad from config - fills empty fields from app.env, app.name, app.tags
	// Existing ModelId values take precedence over config values
	if err := modelId.PadFromConfig(config); err != nil {
		return cfg.AppIdentity{}, fmt.Errorf("failed to pad model id from config: %w", err)
	}

	return cfg.AppIdentity{
		Env:  modelId.Env,
		Name: modelId.App,
		Tags: cfg.AppTags(modelId.Tags),
	}, nil
}
