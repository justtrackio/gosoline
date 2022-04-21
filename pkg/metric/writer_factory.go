package metric

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	WriterTypeCw   = "cw"
	WriterTypeES   = "es"
	WriterTypeProm = "prom"
)

func ProvideMetricWriterByType(ctx context.Context, config cfg.Config, logger log.Logger, typ string) (Writer, error) {
	switch typ {
	case WriterTypeCw:
		return NewCwWriter(ctx, config, logger)
	case WriterTypeES:
		return NewEsWriter(config, logger)
	case WriterTypeProm:
		return NewPromWriter(ctx, config, logger)
	}

	return nil, fmt.Errorf("metric writer type of %s not found", typ)
}
