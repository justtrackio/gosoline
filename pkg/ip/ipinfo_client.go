package ip

import (
	"context"
	"encoding/json"
	"fmt"
	goHttp "net/http"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

const ipInfoUrl = "https://ipinfo.io"

type IpInfoClient struct {
	logger log.Logger
	http   http.Client
}

type IpInfoData struct {
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

func (ii *IpInfoData) toIpData() *Data {
	d := Data(*ii)

	return &d
}

type ipInfoClientCtxKey string

func ProvideIpInfoClient(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error) {
	return appctx.Provide(ctx, ipInfoClientCtxKey("default"), func() (Client, error) {
		return NewIpInfoClient(ctx, config, logger)
	})
}

func NewIpInfoClient(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error) {
	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "ipInfo")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	return &IpInfoClient{logger: logger, http: httpClient}, nil
}

func (c *IpInfoClient) GetIpData(ctx context.Context) (*Data, error) {
	req := c.http.NewRequest().
		WithHeader("Accept", http.MimeTypeApplicationJson).
		WithUrl(ipInfoUrl)

	resp, err := c.http.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not request ipinfo api: %w", err)
	}

	if resp.StatusCode != goHttp.StatusOK {
		return nil, fmt.Errorf("received non-ok status from ipinfo api")
	}

	if len(resp.Body) == 0 {
		return nil, fmt.Errorf("ipinfo response body is empty")
	}

	var info IpInfoData
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return nil, fmt.Errorf("error unmarshalling ip info: %w", err)
	}

	return info.toIpData(), nil
}
