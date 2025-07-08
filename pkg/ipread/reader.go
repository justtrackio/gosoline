package ipread

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

var (
	ErrIpParseFailed = errors.New("failed to parse geo ip")
	ErrIpNotFound    = errors.New("ip not found")
)

type GeoCity struct {
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Ip          string `json:"ip"`
	TimeZone    string `json:"timeZone"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Reader
type Reader interface {
	City(ipString string) (*GeoCity, error)
}

type reader struct {
	provider Provider
}

type readerCtxKey string

func ProvideReader(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*reader, error) {
	return appctx.Provide(ctx, readerCtxKey(name), func() (*reader, error) {
		return NewReader(ctx, config, logger, name)
	})
}

func NewReader(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*reader, error) {
	logger = logger.WithChannel("ipread")
	settings := readSettings(config, name)

	var ok bool
	var err error
	var factory ProviderFactory
	var provider Provider

	if factory, ok = providers[settings.Provider]; !ok {
		return nil, fmt.Errorf("provider %s not found", settings.Provider)
	}

	if provider, err = factory(ctx, config, logger, name); err != nil {
		return nil, fmt.Errorf("can not create ip reader provider: %w", err)
	}

	read := &reader{
		provider: provider,
	}

	return read, nil
}

func (r reader) City(ipString string) (*GeoCity, error) {
	ip := net.ParseIP(ipString)

	if ip == nil {
		return nil, ErrIpParseFailed
	}

	record, err := r.provider.City(ip)
	if err != nil {
		return nil, err
	}

	return &GeoCity{
		Ip:          ipString,
		CountryCode: record.Country.IsoCode,
		City:        record.City.Names["en"],
		TimeZone:    record.Location.TimeZone,
	}, nil
}
