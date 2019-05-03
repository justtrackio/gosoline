package db_test

import (
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/applike/gosoline/pkg/db"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestConnection_Boot(t *testing.T) {
	connection := db.NewConnection()

	dbUserName := "testBoot"
	dbPassword := "123456"
	dbHostName := "database.applike.unittest"
	dbPort := 3306
	dbDatabase := "analytics"

	dbMock, sqlMock := setupSqlMock(dbUserName, dbPassword, dbHostName)

	defer func() {
		err := dbMock.Close()
		assert.Nil(t, err)
	}()

	loggerMock := monMocks.NewLoggerMockedAll()
	settings := getSettings(dbUserName, dbPassword, dbHostName, dbPort, dbDatabase)

	assert.NotPanics(t, func() {
		connection.BootWithInterfaces(loggerMock, settings)
	})

	sqlMock.ExpectClose()
}

func TestConnection_Run_And_Stop(t *testing.T) {
	connection := db.NewConnection()

	dbUserName := "testRunAndStop"
	dbPassword := "123456"
	dbHostName := "database.applike.unittest"
	dbPort := 3306
	dbDatabase := "analytics"

	dbMock, sqlMock := setupSqlMock(dbUserName, dbPassword, dbHostName)

	defer func() {
		err := dbMock.Close()
		assert.Nil(t, err)
	}()

	loggerMock := monMocks.NewLoggerMockedAll()
	settings := getSettings(dbUserName, dbPassword, dbHostName, dbPort, dbDatabase)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() {
		connection.BootWithInterfaces(loggerMock, settings)
		go connection.Run(ctx)
	})

	sqlMock.ExpectClose()
}

func setupSqlMock(user string, password string, hostname string) (sql.DB, sqlmock.Sqlmock) {
	dsn := user + ":" + password + "@tcp(" + hostname + ":3306)/analytics?charset=utf8mb4&multiStatements=true&parseTime=false"
	dbMock, sqlMock, _ := sqlmock.NewWithDSN(dsn)

	return *dbMock, sqlMock
}

func getSettings(dbUserName string, dbPassword string, dbHostName string, dbPort int, dbDatabase string) db.Settings {
	return db.Settings{
		DriverName:         "sqlmock",
		Host:               dbHostName,
		Port:               dbPort,
		Database:           dbDatabase,
		User:               dbUserName,
		Password:           dbPassword,
		RetryDelay:         time.Duration(1 * time.Second),
		ConnectionLifetime: time.Duration(10 * time.Second),
	}
}
