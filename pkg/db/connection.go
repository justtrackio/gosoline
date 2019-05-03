package db

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Settings struct {
	Application string `cfg:"app_name"`

	DriverName string `cfg:"db_drivername"`
	Host       string `cfg:"db_hostname"`
	Port       int    `cfg:"db_port"`
	Database   string `cfg:"db_database"`
	User       string `cfg:"db_username"`
	Password   string `cfg:"db_password"`

	RetryDelay         time.Duration `cfg:"db_retry_wait"`
	ConnectionLifetime time.Duration `cfg:"db_max_connection_lifetime"`

	ParseTime      bool `cfg:"db_parse_time"`
	AutoMigrate    bool `cfg:"db_auto_migrate"`
	PrefixedTables bool `cfg:"db_table_prefixed"`
}

type Connection struct {
	kernel.BackgroundModule

	logger   mon.Logger
	settings Settings
	db       *sqlx.DB

	stop chan struct{}
}

var barrier = sync.WaitGroup{}
var DefaultConnection *sqlx.DB = nil

func NewConnection() *Connection {
	barrier.Add(1)

	connection := Connection{
		stop: make(chan struct{}),
	}

	return &connection
}

func (c *Connection) Boot(config cfg.Config, logger mon.Logger) error {
	defer barrier.Done()

	settings := Settings{}
	config.Bind(&settings)

	return c.BootWithInterfaces(logger, settings)
}

func (c *Connection) BootWithInterfaces(logger mon.Logger, settings Settings) error {
	c.logger = logger
	c.settings = settings

	err := backoff.Retry(func() error {
		err := c.connect()

		if err != nil {
			c.logger.Error(err, "db.init returned error. retrying...")
		}

		return err
	}, backoff.NewConstantBackOff(c.settings.RetryDelay*time.Second))

	if err != nil {
		c.logger.Fatal(err, "db.init returned error")
	}

	c.db.SetConnMaxLifetime(c.settings.ConnectionLifetime * time.Second)
	DefaultConnection = c.db

	c.runMigrations()

	return nil
}

func (c *Connection) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Connection) connect() error {
	var err error

	dsn := url.URL{
		User: url.UserPassword(c.settings.User, c.settings.Password),
		Host: fmt.Sprintf("tcp(%s:%v)", c.settings.Host, c.settings.Port),
		Path: c.settings.Database,
	}

	qry := dsn.Query()
	qry.Set("multiStatements", "true")
	qry.Set("parseTime", strconv.FormatBool(c.settings.ParseTime))
	qry.Set("charset", "utf8mb4")
	dsn.RawQuery = qry.Encode()

	db, err := sqlx.Open(c.settings.DriverName, dsn.String()[2:])

	if err != nil {
		return err
	}

	err = db.Ping()

	if err != nil {
		return err
	}

	c.db = db

	return nil
}

func GenerateVersionedTableName(table string, version int) string {
	return fmt.Sprintf("V%v_%v", version, table)
}
