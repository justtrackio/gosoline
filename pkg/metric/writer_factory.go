package metric

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	writerTypeCloudwatch    = "cw"
	writerTypeElasticsearch = "es"
	writerTypePrometheus    = "prom"
)

type metricWriterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error)

var metricWriterFactories = map[string]metricWriterFactory{
	writerTypeCloudwatch:    ProvideCloudwatchWriter,
	writerTypeElasticsearch: ProvideElasticsearchWriter,
	writerTypePrometheus:    ProvidePrometheusWriter,
}

func NewMetricWriter(ctx context.Context, config cfg.Config, logger log.Logger, types []string) (Writer, error) {
	writers := make([]Writer, 0, len(types))

	for _, typ := range types {
		var w Writer
		var err error

		factory, ok := metricWriterFactories[typ]
		if !ok {
			return nil, fmt.Errorf("unrecognized writer type: %s", typ)
		}

		w, err = factory(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("could not create %s metric writer: %w", typ, err)
		}

		writers = append(writers, w)
	}

	return NewMultiWriterWithInterfaces(writers), nil
}

func RegisterMetricWriterFactory(name string, factory metricWriterFactory) {
	metricWriterFactories[name] = factory
}
