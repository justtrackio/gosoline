package fixtures

import (
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
)

type MysqlPlainFixtureValues []interface{}

type MysqlPlainMetaData struct {
	TableName string
	Columns   []string
}

type mysqlPlainFixtureWriter struct {
	logger   mon.Logger
	client   db.Client
	metadata *MysqlPlainMetaData
	purger   *mysqlPurger
}

func MysqlPlainFixtureWriterFactory(metadata *MysqlPlainMetaData) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) (FixtureWriter, error) {
		dbClient := db.NewClient(config, logger, "default")
		purger := newMysqlPurger(config, logger, metadata.TableName)

		return NewMysqlPlainFixtureWriterWithInterfaces(logger, dbClient, metadata, purger), nil
	}
}

func NewMysqlPlainFixtureWriterWithInterfaces(logger mon.Logger, client db.Client, metadata *MysqlPlainMetaData, purger *mysqlPurger) FixtureWriter {
	return &mysqlPlainFixtureWriter{
		logger:   logger,
		client:   client,
		metadata: metadata,
		purger:   purger,
	}
}

func (m *mysqlPlainFixtureWriter) Purge() error {
	err := m.purger.purgeMysql()

	if err != nil {
		m.logger.Errorf(err, "error occured during purging of table %s in plain mysql fixture loader", m.metadata.TableName)

		return err
	}

	m.logger.Infof("purged table %s for plain mysql fixtures", m.metadata.TableName)

	return nil
}

func (m *mysqlPlainFixtureWriter) Write(fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
		fixture := item.(MysqlPlainFixtureValues)

		sql, args, err := m.buildSql(fixture)

		if err != nil {
			return err
		}

		res, err := m.client.Exec(sql, args...)

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

func (m *mysqlPlainFixtureWriter) buildSql(values MysqlPlainFixtureValues) (string, []interface{}, error) {
	insertBuilder := squirrel.Replace(m.metadata.TableName).
		PlaceholderFormat(squirrel.Question).
		Columns(m.metadata.Columns...).
		Values(values...)

	return insertBuilder.ToSql()
}
