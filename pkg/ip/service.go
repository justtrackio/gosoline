package ip

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type dataCtxKey string

const configKey = "ip_provider_settings"

type ProviderChainConfig struct {
	Providers []string `cfg:"providers" json:"providers" default:"ipinfo,ifconfig"`
}

type Data struct {
	Ip       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	Readme   string `json:"readme"`
}

func ProvideData(ctx context.Context, config cfg.Config, logger log.Logger) (*Data, error) {
	return appctx.Provide(ctx, dataCtxKey("ip_data"), func() (*Data, error) {
		client, err := ProvideChainClient(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("could not create client: %w", err)
		}

		return client.GetIpData(ctx)
	})
}
