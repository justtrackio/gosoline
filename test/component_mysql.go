package test

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"log"
)

const mysqlContainerName = "gosoline_test_mysql"

var IntegrationTestDb *sql.DB

type mysqlConfig struct {
	Version string `mapstructure:"version"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	DbName  string `mapstructure:"dbName"`
}

type noopLogger struct {
}

func (l noopLogger) Print(v ...interface{}) {
}

func init() {
	err = mysql.SetLogger(&noopLogger{})

	if err != nil {
		panic(err)
	}
}

func runMysql(name string, config configInput) {
	wait.Add(1)
	go doRunMysql(name, config)
}

func doRunMysql(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "mysql")

	config := &mysqlConfig{}
	unmarshalConfig(configMap, config)
	runMysqlContainer(config)
}

func runMysqlContainer(config *mysqlConfig) {
	env := []string{
		fmt.Sprintf("MYSQL_DATABASE=%s", config.DbName),
		"MYSQL_USER=gosoline",
		"MYSQL_PASSWORD=gosoline",
		"MYSQL_ROOT_PASSWORD=gosoline",
	}

	runContainer(mysqlContainerName, ContainerConfig{
		Repository: "mysql",
		Tag:        config.Version,
		Env:        env,
		Cmd:        []string{"--sql_mode=NO_ENGINE_SUBSTITUTION"},
		PortBindings: PortBinding{
			"3306/tcp": fmt.Sprint(config.Port),
		},
		HealthCheck: func() error {
			dsn := fmt.Sprintf("%s:%s@(%s:%d)/%s?parseTime=true", "gosoline", "gosoline", config.Host, config.Port, config.DbName)
			IntegrationTestDb, err = sql.Open("mysql", dsn)

			if err != nil {
				return errors.Wrapf(err, "can not open mysql connection %s", dsn)
			}

			err = IntegrationTestDb.Ping()

			if err != nil {
				return errors.Wrapf(err, "can not ping mysql connection %s", dsn)
			}

			return nil
		},
	})
}
