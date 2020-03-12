// +build fixtures

package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type fixtureLoaderSettings struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type fixtureLoader struct {
	logger        mon.Logger
	settings      *fixtureLoaderSettings
	writerFactory func(factory FixtureWriterFactory) FixtureWriter
}

func NewFixtureLoader(config cfg.Config, logger mon.Logger) FixtureLoader {
	logger = logger.WithChannel("fixture_loader")

	settings := &fixtureLoaderSettings{}
	config.UnmarshalKey("fixtures", settings)

	return &fixtureLoader{
		logger:   logger,
		settings: settings,
		writerFactory: func(factory FixtureWriterFactory) FixtureWriter {
			return factory(config, logger)
		},
	}
}

func (f *fixtureLoader) Load(fixtureSets []*FixtureSet) error {
	if !f.settings.Enabled {
		f.logger.Info("fixture loader is not enabled")
		return nil
	}

	for _, fs := range fixtureSets {

		if !fs.Enabled {
			f.logger.Info("skipping disabled fixture set")
			continue
		}

		if fs.Writer == nil {
			return fmt.Errorf("fixture set is missing a writer")
		}

		writer := f.writerFactory(fs.Writer)

		if fs.Purge {
			err := writer.Purge()

			if err != nil {
				return fmt.Errorf("error during purging of fixture set: %w", err)
			}
		}

		err := writer.Write(fs)

		if err != nil {
			return fmt.Errorf("error during loading of fixture set: %w", err)
		}
	}

	return nil
}
