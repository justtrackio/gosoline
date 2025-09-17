package ip

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Factory func(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error)

type Client interface {
	GetIpData(ctx context.Context) (*Data, error)
}

var clientFactories = map[string]Factory{
	ProviderIpInfo:   ProvideIpInfoClient,
	ProviderIfconfig: ProvideIfConfigClient,
}
