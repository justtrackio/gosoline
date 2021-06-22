package metric

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
)

const (
	WriterTypeCw = "cw"
	WriterTypeES = "es"
)

func ProvideMetricWriterByType(config cfg.Config, logger log.Logger, typ string) (Writer, error) {
	switch typ {
	case WriterTypeCw:
		return NewCwWriter(config, logger)
	case WriterTypeES:
		return NewEsWriter(config, logger)
	}

	return nil, fmt.Errorf("metric writer type of %s not found", typ)
}
