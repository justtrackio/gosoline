package ip

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ChainClient struct {
	logger log.Logger
	chain  []namedClient
}

type namedClient struct {
	name   string
	client Client
}

type clientCtxKey string

func ProvideChainClient(ctx context.Context, config cfg.Config, logger log.Logger) (*ChainClient, error) {
	return appctx.Provide(ctx, clientCtxKey("default"), func() (*ChainClient, error) {
		return newChainClient(ctx, config, logger)
	})
}

func newChainClient(ctx context.Context, config cfg.Config, logger log.Logger) (*ChainClient, error) {
	var chainConfig ProviderChainConfig
	if err := config.UnmarshalKey(configKey, &chainConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal provider-chain-config: %w", err)
	}

	if len(chainConfig.Providers) == 0 {
		return nil, fmt.Errorf("no ip providers configured")
	}

	chainClient := &ChainClient{
		logger: logger,
		chain:  make([]namedClient, 0, len(chainConfig.Providers)),
	}

	for _, provider := range chainConfig.Providers {
		clientFactory, ok := clientFactories[provider]
		if !ok {
			return nil, fmt.Errorf("unknown ip provider %s", provider)
		}

		client, err := clientFactory(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to build client for ip provider %s: %w", provider, err)
		}

		chainClient.chain = append(chainClient.chain, namedClient{
			name:   provider,
			client: client,
		})
	}

	return chainClient, nil
}

func (c *ChainClient) GetIpData(ctx context.Context) (*Data, error) {
	if len(c.chain) == 0 {
		return nil, fmt.Errorf("no ip providers configured")
	}

	multiErr := &multierror.Error{}

	for _, nc := range c.chain {
		data, err := nc.client.GetIpData(ctx)
		if err == nil {
			return data, nil
		}

		c.logger.Warn(ctx, "ip provider %s failed: %w", nc.name, err)

		multiErr = multierror.Append(multiErr, fmt.Errorf("ip provider %s failed: %w", nc.name, err))
	}

	if err := multiErr.ErrorOrNil(); err != nil {
		return nil, fmt.Errorf("all ip providers failed: %w", err)
	}

	return nil, nil
}
