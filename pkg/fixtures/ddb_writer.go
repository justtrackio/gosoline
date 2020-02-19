package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"reflect"
)

type dynamoDbFixtureWriter struct {
	config cfg.Config
	logger mon.Logger
}

func NewDynamoDbFixtureWriter(cfg cfg.Config, logger mon.Logger) FixtureWriter {
	return cachedWriters.New("dynamoDb", func() FixtureWriter {
		return &dynamoDbFixtureWriter{
			config: cfg,
			logger: logger,
		}
	})
}

func (d *dynamoDbFixtureWriter) WriteFixtures(fs *FixtureSet) error {
	modelId, ok := fs.WriterMetadata.(mdl.ModelId)

	if !ok {
		return fmt.Errorf("invalid writer metadata type: %s", reflect.TypeOf(fs.WriterMetadata))
	}

	if len(fs.Fixtures) == 0 {
		d.logger.Info("loaded 0 dynamo db fixtures")
		return nil
	}

	repo := ddb.NewRepository(d.config, d.logger, &ddb.Settings{
		ModelId: modelId,
		Main: ddb.MainSettings{
			Model:              fs.Fixtures[0], // to extract the metadata only
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	})

	for _, fixture := range fs.Fixtures {
		_, err := repo.PutItem(context.Background(), nil, fixture)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamo db fixtures", len(fs.Fixtures))

	return nil
}
