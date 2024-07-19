package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	FixtureSetBuilders["db"] = MysqlPlainFixtureSetBuilder
}

func MysqlPlainFixtureSetBuilder(ctx context.Context, config cfg.Config, logger log.Logger, settings FixtureSetBuilderSettings) ([]*FixtureSet, error) {
	logger = logger.WithChannel("mysql-plain-fixture-set-builder")

	fixturesByType := settings.Fixtures
	fsDbName := settings.DbName
	fss := make([]*FixtureSet, 0, len(fixturesByType))

	if !settings.Enabled {
		return fss, nil
	}

	stateService, err := provideMysqlStateService(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create state service: %w", err)
	}

	for tableName, fixtures := range fixturesByType {
		fixtureState, err := stateService.Get(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get fixture state: %w", err)
		}

		if fixtureState != nil && fixtureState.DataSetDbName == fsDbName {
			logger.Info("skipping fixture set builder for table %s, deployed fixture set matches desired fixture set", tableName)

			continue
		}

		columns := make([]string, 0)
		if len(fixtures) > 0 {
			for col := range fixtures[0] {
				columns = append(columns, col)
			}
		}
		metadata := &MysqlPlainMetaData{
			TableName: tableName,
			Columns:   columns,
		}
		fsFixtures := make([]any, len(fixtures))

		for i, fixture := range fixtures {
			values := make([]any, len(columns))

			for j, column := range columns {
				values[j] = fixture[column]
			}

			fsFixtures[i] = MysqlPlainFixtureValues(values)
		}

		fs := &FixtureSet{
			Enabled:        true,
			Purge:          true,
			Writer:         MysqlPlainFixtureWriterFactory(metadata),
			Fixtures:       fsFixtures,
			FixtureSetName: fsDbName,
		}
		fss = append(fss, fs)
	}

	return fss, nil
}
