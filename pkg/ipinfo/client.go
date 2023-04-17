package ipinfo

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

type Client struct {
	logger log.Logger
	http   http.Client
}

type IpInfo struct {
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

type ipInfoClientCtxKey string

type ipInfoCtxKey string

func ProvideIpInfo(ctx context.Context, config cfg.Config, logger log.Logger) (*IpInfo, error) {
	return appctx.Provide(ctx, ipInfoCtxKey("ip"), func() (*IpInfo, error) {
		c, err := ProvideClient(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return c.GetIpInfo(ctx)
	})
}

func ProvideClient(ctx context.Context, config cfg.Config, logger log.Logger) (*Client, error) {
	return appctx.Provide(ctx, ipInfoClientCtxKey("default"), func() (*Client, error) {
		return NewClient(ctx, config, logger)
	})
}

func NewClient(ctx context.Context, config cfg.Config, logger log.Logger) (*Client, error) {
	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "ipInfo")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	return &Client{logger: logger, http: httpClient}, nil
}

func (c *Client) GetIpInfo(ctx context.Context) (*IpInfo, error) {
	req := c.http.NewRequest().
		WithHeader("Accept", http.MimeTypeApplicationJson).
		WithUrl(ipInfoUrl)

	resp, err := c.http.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not request api: %w", err)
	}

	if resp.StatusCode != goHttp.StatusOK {
		return nil, fmt.Errorf("received non-ok status")
	}

	if len(resp.Body) == 0 {
		return nil, fmt.Errorf("body is empty")
	}

	var info IpInfo
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return nil, fmt.Errorf("error unmarshalling ip info: %w", err)
	}

	return &info, nil
}
