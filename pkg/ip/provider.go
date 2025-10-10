package ip

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ProviderIpInfo   = "ipinfo"
	ProviderIfconfig = "ifconfig"
)

var providers = map[string]Factory{
	ProviderIpInfo:   ProvideIpInfoClient,
	ProviderIfconfig: ProvideIfConfigClient,
}

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger, providerConfig ProviderConfig) (Client, error) {
	var err error
	client := &fallbackClient{
		logger: logger,
	}

	makePrimaryClient, ok := providers[providerConfig.Primary]
	if !ok {
		return nil, fmt.Errorf("unknown primary ip provider %s", providerConfig.Primary)
	}

	client.primary, err = makePrimaryClient(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build primary ip client for provider %s: %w", providerConfig.Primary, err)
	}

	if providerConfig.FallbackTo == "" || providerConfig.FallbackTo == providerConfig.Primary {
		return client.primary, nil
	}

	makeFallbackClient, ok := providers[providerConfig.FallbackTo]
	if !ok {
		logger.Warn(ctx, "unknown fallback ip provider %q", providerConfig.FallbackTo)

		return client.primary, nil
	}

	client.fallback, err = makeFallbackClient(ctx, config, logger)
	if err != nil {
		logger.Warn(ctx, "failed to build fallback ip client for provider %s: %v", providerConfig.FallbackTo, err)

		return client.primary, nil
	}

	return client, nil
}
