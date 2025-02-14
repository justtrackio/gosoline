package metric

import (
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	WithPromNSNamingStrategy(func(modelId cfg.AppId) string {
		return replacer.Replace(modelId.String())
	})
}

var replacer = strings.NewReplacer("-", "_")

type NamingFactory func(modelId cfg.AppId) string

var promNSNamingStrategy NamingFactory

func WithPromNSNamingStrategy(strategy NamingFactory) {
	promNSNamingStrategy = strategy
}
