package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

func ApiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	definitions := &httpserver.Definitions{}

	euroHandler, err := NewEuroHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroHandler: %w", err)
	}

	euroAtDateHandler, err := NewEuroAtDateHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroAtDateHandler: %w", err)
	}

	definitions.GET("/euro/:amount/:currency", httpserver.CreateHandler(euroHandler))
	definitions.GET("/euro-at-date/:amount/:currency/:date", httpserver.CreateHandler(euroAtDateHandler))

	return definitions, nil
}
