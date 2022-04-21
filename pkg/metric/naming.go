package metric

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type NamingFactory func(modelId cfg.AppId) string

var promNSNamingStrategy = func(modelId cfg.AppId) string {
	return fmt.Sprintf("%v:%v:%v:%v", modelId.Project, modelId.Environment, modelId.Family, modelId.Application)
}

func WithPromNSNamingStrategy(strategy NamingFactory) {
	promNSNamingStrategy = strategy
}
