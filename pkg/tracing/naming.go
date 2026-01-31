package tracing

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type NamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}-{app.name}"`
}

func resolveAppId(config cfg.Config) (string, error) {
	namingSettings := &NamingSettings{}
	if err := config.UnmarshalKey("tracing.naming", namingSettings); err != nil {
		return "", fmt.Errorf("failed to unmarshal tracing naming settings: %w", err)
	}

	appId, err := config.FormatString(namingSettings.Pattern)
	if err != nil {
		return "", fmt.Errorf("failed to format service name: %w", err)
	}

	return appId, nil
}
