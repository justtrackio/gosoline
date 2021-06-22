package ipread

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/oschwald/geoip2-golang"
	"net"
)

type MemoryRecord struct {
	CountryIso string `cfg:"country_iso"`
	CityName   string `cfg:"city_name"`
	TimeZone   string `cfg:"time_zone"`
}

type memoryProvider struct {
	records map[string]*geoip2.City
}

var memoryProviderContainer = make(map[string]*memoryProvider)

func ProvideMemoryProvider(name string) *memoryProvider {
	if _, ok := memoryProviderContainer[name]; ok {
		return memoryProviderContainer[name]
	}

	memoryProviderContainer[name] = &memoryProvider{
		records: make(map[string]*geoip2.City),
	}

	return memoryProviderContainer[name]
}

func NewMemoryProvider(_ cfg.Config, _ log.Logger, name string) (Provider, error) {
	return ProvideMemoryProvider(name), nil
}

func (p memoryProvider) City(ipAddress net.IP) (*geoip2.City, error) {
	ipString := ipAddress.String()

	if _, ok := p.records[ipString]; !ok {
		return nil, ErrIpNotFound
	}

	return p.records[ipString], nil
}

func (p memoryProvider) AddRecord(ipString string, record MemoryRecord) {
	p.records[ipString] = &geoip2.City{
		City: struct {
			GeoNameID uint              `maxminddb:"geoname_id"`
			Names     map[string]string `maxminddb:"names"`
		}{
			Names: map[string]string{
				"en": record.CityName,
			},
		},
		Country: struct {
			GeoNameID         uint              `maxminddb:"geoname_id"`
			IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
			IsoCode           string            `maxminddb:"iso_code"`
			Names             map[string]string `maxminddb:"names"`
		}{
			IsoCode: record.CountryIso,
		},
		Location: struct {
			AccuracyRadius uint16  `maxminddb:"accuracy_radius"`
			Latitude       float64 `maxminddb:"latitude"`
			Longitude      float64 `maxminddb:"longitude"`
			MetroCode      uint    `maxminddb:"metro_code"`
			TimeZone       string  `maxminddb:"time_zone"`
		}{
			TimeZone: record.TimeZone,
		},
	}
}
