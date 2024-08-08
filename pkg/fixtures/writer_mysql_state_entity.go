package fixtures

import "github.com/justtrackio/gosoline/pkg/db-repo"

func newMysqlStateFixture(tableName string, dataSetDbName string) *mysqlStateFixture {
	return &mysqlStateFixture{
		DataSetDbName:  dataSetDbName,
		LocalTableName: tableName,
	}
}

type mysqlStateFixture struct {
	db_repo.Model
	LocalTableName string
	DataSetDbName  string
}

func (f *mysqlStateFixture) TableName() string {
	return "fixture"
}
