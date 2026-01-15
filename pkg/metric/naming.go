package metric

import (
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	WithPromNSNamingStrategy(func(identity cfg.AppIdentity) string {
		return replacer.Replace(identity.String())
	})
}

var replacer = strings.NewReplacer("-", "_")

type NamingFactory func(identity cfg.AppIdentity) string

var promNSNamingStrategy NamingFactory

func WithPromNSNamingStrategy(strategy NamingFactory) {
	promNSNamingStrategy = strategy
}
