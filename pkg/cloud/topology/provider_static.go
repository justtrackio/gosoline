package topology

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// StaticSettings holds the configuration for the static topology provider.
type StaticSettings struct {
	// Zone is the static topology zone identifier to return (e.g., an availability zone).
	Zone string `cfg:"zone"`
}

type staticProvider struct {
	zone string
}

func newStaticProvider(_ context.Context, _ cfg.Config, _ log.Logger, settings Settings) (Provider, error) {
	return NewStaticProviderWithInterfaces(settings)
}

// NewStaticProviderWithInterfaces creates a static topology provider with explicit dependencies.
// This is useful for testing without a full application context.
func NewStaticProviderWithInterfaces(settings Settings) (Provider, error) {
	if settings.Static.Zone == "" {
		return nil, fmt.Errorf("static topology zone is not configured: set %s.static.zone", configKey)
	}

	return &staticProvider{
		zone: settings.Static.Zone,
	}, nil
}

func (p *staticProvider) GetZone(_ context.Context) (string, error) {
	return p.zone, nil
}
