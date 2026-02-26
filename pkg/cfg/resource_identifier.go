package cfg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// ResourceIdentifier identifies a remote resource (e.g. a queue, topic, or stream)
// by the application that owns it, together with the environment and tags that are
// used to resolve the resource's naming pattern.
//
// It is designed to be embedded into stream input/output configuration structs so
// that the config keys are flat (no extra nesting level):
//
//	type sqsInputConfiguration struct {
//	    cfg.ResourceIdentifier
//	    QueueId string `cfg:"queue_id"`
//	    ...
//	}
//
// Example YAML:
//
//	stream:
//	  input:
//	    my-input:
//	      type: sqs
//	      application: user-service   # the APPLICATION that owns the queue
//	      env: prod                   # optional, defaults to app.env
//	      tags:                       # optional, merged with app.tags
//	        project: my-project
//	      queue_id: user-events       # the resource-specific identifier
//
// The separation between ResourceIdentifier and the resource key (queue_id,
// topic_id, stream_name, …) is intentional: "application" is unambiguously the
// owning app name, while the resource key describes what to consume/produce.
//
// To obtain a cfg.Identity suitable for the existing naming functions call
// ToIdentity() after PadFromConfig().
type ResourceIdentifier struct {
	// Env is the environment of the owning application (e.g. "prod", "dev").
	// Defaults to app.env when empty.
	Env string `cfg:"env"`

	// Application is the name of the application that owns the resource.
	// Defaults to app.name when empty.
	// This maps to {app.name} in naming patterns.
	Application string `cfg:"application"`

	// Tags are the tags of the owning application used for pattern expansion.
	// Missing keys are filled from app.tags; existing keys are preserved.
	Tags Tags `cfg:"tags"`
}

// PadFromConfig fills empty fields of ResourceIdentifier from global app config.
//
// Behaviour mirrors Identity.PadFromConfig:
//   - If Application is empty, fills from app.name (required, errors if missing/empty)
//   - If Env is empty, fills from app.env (required, errors if missing/empty)
//   - Tags are merged with app.tags; per-resource tag values win over app-level ones
func (r *ResourceIdentifier) PadFromConfig(config Config) error {
	var err error
	var tags map[string]string

	if r.Application == "" {
		if r.Application, err = config.GetString("app.name"); err != nil {
			return fmt.Errorf("app.name: %w", err)
		}

		r.Application = strings.TrimSpace(r.Application)

		if r.Application == "" {
			return errors.New("app.name: value is empty")
		}
	}

	if r.Env == "" {
		if r.Env, err = config.GetString("app.env"); err != nil {
			return fmt.Errorf("app.env: %w", err)
		}

		r.Env = strings.TrimSpace(r.Env)

		if r.Env == "" {
			return errors.New("app.env: value is empty")
		}
	}

	if tags, err = config.GetStringMapString("app.tags", map[string]string{}); err != nil {
		return fmt.Errorf("app.tags: %w", err)
	}

	r.Tags = funk.MergeMaps(tags, r.Tags)

	return nil
}

// ToIdentity converts the ResourceIdentifier into a cfg.Identity, mapping
// Application → Identity.Name. The returned Identity has no Namespace set;
// call Identity.PadFromConfig to populate it from app.namespace when needed.
func (r ResourceIdentifier) ToIdentity() Identity {
	return Identity{
		Env:  r.Env,
		Name: r.Application,
		Tags: r.Tags,
	}
}
