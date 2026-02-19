package tracing

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type NamingSettings struct {
	Pattern   string `cfg:"pattern,nodecode" default:"{app.namespace}-{app.name}"`
	Delimiter string `cfg:"delimiter" default:"-"`
}

func resolveAppId(config cfg.Config) (string, error) {
	var err error
	var identity cfg.Identity
	var appId string

	namingSettings := &NamingSettings{}
	if err := config.UnmarshalKey("tracing.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal tracing naming settings: %w", err)
	}

	if identity, err = cfg.GetAppIdentity(config); err != nil {
		return "", fmt.Errorf("could not get app identity from config: %w", err)
	}

	if appId, err = identity.Format(namingSettings.Pattern, namingSettings.Delimiter); err != nil {
		return "", fmt.Errorf("failed to format service name: %w", err)
	}

	return appId, nil
}
