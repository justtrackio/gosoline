package topology

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const configKey = "cloud.topology"

// Provider resolves the topology zone for the current instance (e.g., an availability zone).
//
//go:generate go run github.com/vektra/mockery/v2 --name Provider
type Provider interface {
	// GetZone returns the topology zone identifier (typically an availability zone) for the current instance.
	// Returns an empty string if no topology information is available (e.g., noop provider).
	GetZone(ctx context.Context) (string, error)
}

// ProviderFactory creates a topology Provider from configuration.
type ProviderFactory func(ctx context.Context, config cfg.Config, logger log.Logger, settings Settings) (Provider, error)

var providerFactories = map[string]ProviderFactory{
	"ec2":    newEc2Provider,
	"static": newStaticProvider,
}

// SetProviderFactory registers a custom topology provider factory under the given name.
// This allows users to extend the topology system with custom implementations that can
// be selected via the cloud.topology.provider configuration key.
func SetProviderFactory(name string, factory ProviderFactory) {
	providerFactories[name] = factory
}

// Settings holds the configuration for the topology provider.
type Settings struct {
	// Provider selects which topology provider to use. Built-in options are "ec2" and "static".
	// Custom providers can be registered via SetProviderFactory. An empty value means topology is disabled.
	Provider string `cfg:"provider"`
	// Static holds the configuration for the static topology provider.
	Static StaticSettings `cfg:"static"`
}

// ProvideProvider returns a singleton topology Provider based on configuration.
// If no provider is configured (Settings.Provider is empty), a noop provider is returned
// whose GetZone method returns an empty string and no error.
func ProvideProvider(ctx context.Context, config cfg.Config, logger log.Logger) (Provider, error) {
	return appctx.Provide(ctx, providerKey{}, func() (Provider, error) {
		return NewProvider(ctx, config, logger)
	})
}

type providerKey struct{}

// NewProvider creates a topology Provider from configuration without singleton semantics.
// Use ProvideProvider for singleton behavior.
func NewProvider(ctx context.Context, config cfg.Config, logger log.Logger) (Provider, error) {
	var settings Settings
	if err := config.UnmarshalKey(configKey, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal topology settings: %w", err)
	}

	if settings.Provider == "" {
		return &noopProvider{}, nil
	}

	factory, ok := providerFactories[settings.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown topology provider %q: register it via topology.SetProviderFactory", settings.Provider)
	}

	return factory(ctx, config, logger, settings)
}

// noopProvider is returned when no topology provider is configured.
// Its GetZone method returns an empty string and no error.
type noopProvider struct{}

func (p *noopProvider) GetZone(_ context.Context) (string, error) {
	return "", nil
}
