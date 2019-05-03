package mon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
)

const (
	MetricWriterTypeCw = "cw"
	MetricWriterTypeES = "es"
)

func ProvideMetricWriterByType(config cfg.Config, logger Logger, typ string) MetricWriter {
	switch typ {
	case MetricWriterTypeCw:
		return NewMetricCwWriter(config, logger)
	case MetricWriterTypeES:
		return NewMetricEsWriter(config, logger)
	}

	logger.Fatalf(fmt.Errorf("unknown metric writer type"), "metric writer type of %s not found", typ)

	return nil
}
