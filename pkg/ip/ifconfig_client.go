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

const ifConfigUrl = "https://ifconfig.co/json"

type IfConfigClient struct {
	logger log.Logger
	http   http.Client
}

type IfConfigData struct {
	Ip         string  `json:"ip"`
	Hostname   string  `json:"hostname"`
	City       string  `json:"city"`
	RegionName string  `json:"region_name"`
	CountryIso string  `json:"country_iso"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	AsnOrg     string  `json:"asn_org"`
	ZipCode    string  `json:"zip_code"`
	Timezone   string  `json:"time_zone"`
}

func (icd *IfConfigData) toIpData() *Data {
	return &Data{
		Ip:       icd.Ip,
		Hostname: icd.Hostname,
		City:     icd.City,
		Region:   icd.RegionName,
		Country:  icd.CountryIso,
		Loc:      fmt.Sprintf("%.6f,%.6f", icd.Latitude, icd.Longitude),
		Org:      icd.AsnOrg,
		Postal:   icd.ZipCode,
		Timezone: icd.Timezone,
		Readme:   "",
	}
}

type ifConfigClientCtxKey string

func ProvideIfConfigClient(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error) {
	return appctx.Provide(ctx, ifConfigClientCtxKey("default"), func() (Client, error) {
		return NewIfConfigClient(ctx, config, logger)
	})
}

func NewIfConfigClient(ctx context.Context, config cfg.Config, logger log.Logger) (Client, error) {
	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "ifConfig")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	return &IfConfigClient{logger: logger, http: httpClient}, nil
}

func (c *IfConfigClient) GetIpData(ctx context.Context) (*Data, error) {
	req := c.http.NewRequest().
		WithUrl(ifConfigUrl).
		WithHeader("Accept", "application/json").
		WithHeader("User-Agent", "gosoline-ifconfig-client")

	resp, err := c.http.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not request ifconfig api: %w", err)
	}

	if resp.StatusCode != goHttp.StatusOK {
		return nil, fmt.Errorf("received non-ok status from ifconfig api: %d", resp.StatusCode)
	}

	if len(resp.Body) == 0 {
		return nil, fmt.Errorf("ifconfig response body is empty")
	}

	var data IfConfigData
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return nil, fmt.Errorf("error unmarshalling ip data from ifconfig response: %w", err)
	}

	return data.toIpData(), nil
}
