package daemon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	MetricWriterTypeCw = "cw"
	MetricWriterTypeES = "es"
)

func ProvideMetricWriterByType(config cfg.Config, logger mon.Logger, typ string) mon.MetricWriter {
	switch typ {
	case MetricWriterTypeCw:
		return NewMetricCwWriter(config, logger)
	case MetricWriterTypeES:
		return NewMetricEsWriter(config, logger)
	}

	logger.Fatalf(fmt.Errorf("unknown metric writer type"), "metric writer type of %s not found", typ)

	return nil
}
