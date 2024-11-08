package metric

import (
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

var replacer = strings.NewReplacer("-", "_")

type NamingFactory func(modelId cfg.AppId) string

var promNSNamingStrategy = func(modelId cfg.AppId) string {
	return replacer.Replace(modelId.String())
}

func WithPromNSNamingStrategy(strategy NamingFactory) {
	promNSNamingStrategy = strategy
}
