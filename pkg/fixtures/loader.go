// +build fixtures

package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func NewFixtureLoader(config cfg.Config, logger mon.Logger) FixtureLoader {
	logger = logger.WithChannel("fixture_loader")

	return &fixtureLoader{
		config: config,
		logger: logger,
	}
}

type fixtureLoader struct {
	config cfg.Config
	logger mon.Logger
}

type fixtureLoaderSettings struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

func (f *fixtureLoader) Load(fixtureSets []*FixtureSet) error {
	settings := fixtureLoaderSettings{}
	f.config.UnmarshalKey("fixtures", &settings)

	if !settings.Enabled {
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

		writer := fs.Writer(f.config, f.logger)
		err := writer.Write(fs)

		if err != nil {
			return fmt.Errorf("error during loading of fixture set: %w", err)
		}
	}

	return nil
}
