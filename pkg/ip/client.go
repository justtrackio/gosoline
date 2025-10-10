package ip

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Factory func(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error)

type Client interface {
	GetIpData(ctx context.Context) (*Data, error)
}

type fallbackClient struct {
	primary  Client
	fallback Client
	logger   log.Logger
}

func (f *fallbackClient) GetIpData(ctx context.Context) (*Data, error) {
	data, err := f.primary.GetIpData(ctx)
	if err == nil {
		return data, nil
	}

	f.logger.Warn(ctx, "ip primary provider failed: %v", err)

	if f.fallback == nil {
		return nil, fmt.Errorf("failed to fetch ip data")
	}

	data, err = f.fallback.GetIpData(ctx)
	if err != nil {
		return nil, fmt.Errorf("ip fallback provider failed: %w", err)
	}

	return data, nil
}
