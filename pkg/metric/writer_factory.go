package metric

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	WriterTypeCloudwatch    = "cw"
	WriterTypeElasticsearch = "es"
	WriterTypePrometheus    = "prom"
)

func NewMetricWriter(ctx context.Context, config cfg.Config, logger log.Logger, types []string) (Writer, error) {
	writers := make([]Writer, 0, len(types))

	for _, typ := range types {
		var w Writer
		var err error

		switch typ {
		case WriterTypeCloudwatch:
			w, err = ProvideCloudwatchWriter(ctx, config, logger)
		case WriterTypeElasticsearch:
			w, err = ProvideElasticsearchWriter(ctx, config, logger)
		case WriterTypePrometheus:
			w, err = ProvidePrometheusWriter(ctx, config, logger)
		default:
			return nil, fmt.Errorf("unrecognized writer type: %s", typ)
		}
		if err != nil {
			return nil, fmt.Errorf("could not create %s metric writer: %w", typ, err)
		}

		writers = append(writers, w)
	}

	return NewMultiWriterWithInterfaces(writers), nil
}
