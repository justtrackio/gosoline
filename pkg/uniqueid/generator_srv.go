package uniqueid

import (
	"context"
	"fmt"
	"net"
	netHttp "net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

type generatorSrv struct {
	client http.Client
	url    string
}

var srvNamingStrategy = func(appId cfg.AppId) string {
	return fmt.Sprintf("unique-id.%s.%s", appId.Environment, appId.Family)
}

// NewGeneratorSrv use this to fetch unique ids remotely via a service discovery entry
func NewGeneratorSrv(ctx context.Context, config cfg.Config, logger log.Logger) (Generator, error) {
	client := http.NewHttpClient(config, logger)

	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	address := srvNamingStrategy(appId)

	_, srvs, err := net.LookupSRV("", "", address)
	if err != nil {
		return nil, fmt.Errorf("could not lookup srv query for address %s: %w", address, err)
	}

	if len(srvs) == 0 {
		return nil, fmt.Errorf("could not find any unique-id instances")
	}

	url := fmt.Sprintf("http://%s:%d/nextId", srvs[0].Target, srvs[0].Port)

	return &generatorSrv{
		client: client,
		url:    url,
	}, nil
}

func (g *generatorSrv) NextId(ctx context.Context) (*int64, error) {
	out := &NextIdResponse{}

	req := g.client.NewRequest().WithUrl(g.url)

	res, err := g.client.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not make uniqueid request: %w", err)
	}

	if res.StatusCode != netHttp.StatusOK {
		return nil, fmt.Errorf("invalid status code received on unique id request")
	}

	err = json.Unmarshal(res.Body, out)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal uniqueid response: %w", err)
	}

	return &out.Id, nil
}
