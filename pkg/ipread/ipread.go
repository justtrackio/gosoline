package ipread

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"net"
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

type ReaderSettings struct {
	Provider string `cfg:"provider" default:"maxmind"`
}

//go:generate mockery -name Reader
type Reader interface {
	City(ipString string) (*GeoCity, error)
}

type reader struct {
	provider Provider
}

func NewReader(config cfg.Config, logger log.Logger, name string) (*reader, error) {
	key := fmt.Sprintf("ipread.%s", name)
	settings := &ReaderSettings{}
	config.UnmarshalKey(key, settings)

	var ok bool
	var factory ProviderFactory

	if factory, ok = providers[settings.Provider]; !ok {
		return nil, fmt.Errorf("provider %s not found", settings.Provider)
	}

	provider, err := factory(config, logger, name)

	if err != nil {
		return nil, fmt.Errorf("can not create ip reader provider: %w", err)
	}

	reader := &reader{
		provider: provider,
	}

	return reader, nil
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
