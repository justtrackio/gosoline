package test

import (
	"database/sql"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"log"
)

type mysqlSettings struct {
	*mockSettings
	Port    uint   `cfg:"port"`
	Version string `cfg:"version"`
	DbName  string `cfg:"dbName"`
}

type mysqlComponent struct {
	name     string
	settings *mysqlSettings
	db       *sql.DB
}

func (m *mysqlComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	m.name = name
	m.settings = &mysqlSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, m.settings)
}

func (m *mysqlComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type mysql is ready", m.name)

	env := []string{
		fmt.Sprintf("MYSQL_DATABASE=%s", m.settings.DbName),
		"MYSQL_USER=gosoline",
		"MYSQL_PASSWORD=gosoline",
		"MYSQL_ROOT_PASSWORD=gosoline",
	}

	containerName := fmt.Sprintf("gosoline_test_mysql_%s", m.name)

	runner.Run(containerName, containerConfig{
		Repository: "mysql",
		Tag:        m.settings.Version,
		Env:        env,
		Cmd:        []string{"--sql_mode=NO_ENGINE_SUBSTITUTION"},
		PortBindings: portBinding{
			"3306/tcp": fmt.Sprint(m.settings.Port),
		},
		HealthCheck: func() error {
			dsn := fmt.Sprintf("%s:%s@(%s:%d)/%s?parseTime=true", "gosoline", "gosoline", m.settings.Host, m.settings.Port, m.settings.DbName)

			if m.db == nil {
				db, err := sql.Open("mysql", dsn)

				if err != nil {
					return errors.Wrapf(err, "can not open mysql connection %s", dsn)
				}

				if db != nil {
					m.db = db
				}
			}

			err := m.db.Ping()

			if err != nil {
				return errors.Wrapf(err, "can not ping mysql connection %s", dsn)
			}

			return nil
		},
		PrintLogs: m.settings.Debug,
	})
}

type noopLogger struct {
}

func (l noopLogger) Print(v ...interface{}) {
}

func init() {
	err := mysql.SetLogger(&noopLogger{})

	if err != nil {
		panic(err)
	}
}

func (m *mysqlComponent) ProvideClient(string) interface{} {
	return m.db
}
