package fixtures

import (
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
	"reflect"
)

type MysqlPlainFixtureValues []interface{}

type MysqlPlainMetaData struct {
	TableName string
	Columns   []string
}

type mysqlPlainFixtureWriter struct {
	logger   mon.Logger
	dbClient db.Client
	metaData MysqlPlainMetaData
}

func MysqlPlainFixtureWriterFactory(metaData MysqlPlainMetaData) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		dbClient := db.NewClient(config, logger)
		return NewMysqlPlainFixtureWriterWithInterfaces(logger, dbClient, metaData)
	}
}

func NewMysqlPlainFixtureWriterWithInterfaces(logger mon.Logger, dbClient db.Client, metaData MysqlPlainMetaData) FixtureWriter {
	return &mysqlPlainFixtureWriter{
		logger:   logger,
		dbClient: dbClient,
		metaData: metaData,
	}
}

func (m *mysqlPlainFixtureWriter) Write(fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
		fixture, ok := item.(*MysqlPlainFixtureValues)

		if !ok {
			return fmt.Errorf("invalid fixture type: %s", reflect.TypeOf(item))
		}

		sql, args, err := m.buildSql(fixture)

		if err != nil {
			return err
		}

		res, err := m.dbClient.Exec(sql, args...)

		if err != nil {
			return err
		}

		ar, err := res.RowsAffected()

		if err != nil {
			return err
		}

		m.logger.Info(fmt.Sprintf("affected rows while fixture loading: %d", ar))
	}

	m.logger.Infof("loaded %d plain mysql fixtures", len(fs.Fixtures))

	return nil
}

func (m *mysqlPlainFixtureWriter) buildSql(values *MysqlPlainFixtureValues) (string, []interface{}, error) {
	insertBuilder := squirrel.Replace(m.metaData.TableName).
		PlaceholderFormat(squirrel.Question).
		Columns(m.metaData.Columns...).
		Values(*values...)

	return insertBuilder.ToSql()
}
