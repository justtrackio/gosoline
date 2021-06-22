package ipread

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/oschwald/geoip2-golang"
)

func NewMaxmindProvider(config cfg.Config, _ log.Logger, name string) (Provider, error) {
	key := fmt.Sprintf("ipread.%s.maxmind.database", name)
	database := config.GetString(key)
	geoIpReader, err := geoip2.Open(database)

	if err != nil {
		return nil, fmt.Errorf("could not open geo db: %w", err)
	}

	return geoIpReader, nil
}
