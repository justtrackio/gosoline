package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
)

type dynamoDbFixtureWriter struct {
	config  cfg.Config
	logger  mon.Logger
	modelId *mdl.ModelId
}

func DynamoDbFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
	return func(cfg cfg.Config, logger mon.Logger) FixtureWriter {
		writer := &dynamoDbFixtureWriter{
			config:  cfg,
			logger:  logger,
			modelId: modelId,
		}

		return writer
	}
}

func (d *dynamoDbFixtureWriter) Write(fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		d.logger.Info("loaded 0 dynamo db fixtures")
		return nil
	}

	repo := ddb.NewRepository(d.config, d.logger, &ddb.Settings{
		ModelId: *d.modelId,
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
