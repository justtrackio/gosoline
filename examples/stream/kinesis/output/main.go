package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func main() {
	application.RunModule("producer", newOutputModule)
}

type ExampleRecord struct {
	Id        string `json:"id"`
	SoldItems int    `json:"soldItems"`
}

type outputModule struct {
	logger  log.Logger
	uuidGen uuid.Uuid
	output  stream.Output
	modelId mdl.ModelId
}

func newOutputModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var output stream.Output

	if output, err = stream.NewConfigurableOutput(ctx, config, logger, "exampleRecord"); err != nil {
		return nil, fmt.Errorf("can not create output exampleRecord: %w", err)
	}

	modelId := mdl.ModelId{
		Name: "exampleRecord",
	}
	if err = modelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad model id: %w", err)
	}

	module := &outputModule{
		logger:  logger,
		uuidGen: uuid.New(),
		output:  output,
		modelId: modelId,
	}

	return module, nil
}

func (p outputModule) Run(ctx context.Context) error {
	ticker := clock.NewRealTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.Chan():
			records := make([]stream.WritableMessage, rand.Intn(20))

			for i := range records {
				body, err := json.Marshal(ExampleRecord{
					Id:        p.uuidGen.NewV4(),
					SoldItems: rand.Intn(100),
				})
				if err != nil {
					return fmt.Errorf("failed to marshal record: %w", err)
				}

				records[i] = stream.NewJsonMessage(string(body), mdlsub.CreateMessageAttributes(p.modelId, mdlsub.TypeCreate, 0))
			}

			if err := p.output.Write(ctx, records); err != nil {
				return fmt.Errorf("can not publish %d records: %w", len(records), err)
			}

			p.logger.Info(ctx, "published %d records", len(records))
		}
	}
}
