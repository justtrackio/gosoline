package env

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func init() {
	componentFactories[componentMySql] = new(mysqlFactory)
}

const componentMySql = "mysql"

type mysqlCredentials struct {
	DatabaseName string `cfg:"database_name" default:"gosoline"`
	UserName     string `cfg:"user_name" default:"gosoline"`
	UserPassword string `cfg:"user_password" default:"gosoline"`
	RootPassword string `cfg:"root_password" default:"gosoline"`
}

type mysqlSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	ContainerBindingSettings
	UseExternalContainer bool             `cfg:"use_external_container" default:"false"`
	Version              string           `cfg:"version" default:"8.0"`
	Credentials          mysqlCredentials `cfg:"credentials"`
}

type mysqlFactory struct{}

func (f mysqlFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("db") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentMySql) {
		return nil
	}

	if manager.HasType(componentMySql) {
		return nil
	}

	components := config.GetStringMap("db")

	for name := range components {
		driver := config.Get(fmt.Sprintf("db.%s.driver", name))

		if driver != componentMySql {
			continue
		}

		settings := &mysqlSettings{}
		config.UnmarshalDefaults(settings)

		settings.Type = componentMySql
		settings.Name = name

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("can not add default mysql component: %w", err)
		}
	}

	return nil
}

func (f mysqlFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &mysqlSettings{}
}

func (f mysqlFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig:  f.configureContainer(settings),
			healthCheck:      f.healthCheck(settings),
			shutdownCallback: f.dropDatabase(settings),
		},
	}
}

func (f mysqlFactory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*mysqlSettings)

	if s.UseExternalContainer {
		// when using an external instance we need to generate a new database
		s.Credentials.DatabaseName = uuid.New().NewV4()
	} else {
		// ensure to use a free port for the new container
		s.Port = 0
	}

	env := []string{
		fmt.Sprintf("MYSQL_DATABASE=%s", s.Credentials.DatabaseName),
		fmt.Sprintf("MYSQL_USER=%s", s.Credentials.UserName),
		fmt.Sprintf("MYSQL_PASSWORD=%s", s.Credentials.UserPassword),
		fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", s.Credentials.RootPassword),
		fmt.Sprintf("MYSQL_ROOT_HOST=%s", "%"),
	}

	if len(s.Tmpfs) == 0 {
		s.Tmpfs = append(s.Tmpfs, TmpfsSettings{
			Path: "/var/lib/mysql",
		})
	}

	if s.UseExternalContainer {
		return &containerConfig{
			UseExternalContainer: true,
			ContainerBindings: containerBindings{
				"3306/tcp": containerBinding{
					host: s.Host,
					port: strconv.Itoa(s.Port),
				},
			},
		}
	}

	return &containerConfig{
		Repository: "mysql/mysql-server",
		Tmpfs:      s.Tmpfs,
		Tag:        s.Version,
		Env:        env,
		Cmd:        []string{"--sql_mode=NO_ENGINE_SUBSTITUTION", "--log-bin-trust-function-creators=TRUE"},
		PortBindings: portBindings{
			"3306/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
	}
}

func (f mysqlFactory) healthCheck(settings interface{}) ComponentHealthCheck {
	return func(container *container) error {
		s := settings.(*mysqlSettings)
		binding := container.bindings["3306/tcp"]
		client, err := f.connection(s, binding)
		if err != nil {
			return fmt.Errorf("can not create client: %w", err)
		}

		return client.Ping()
	}
}

func (f mysqlFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	s := settings.(*mysqlSettings)
	binding := containers["main"].bindings["3306/tcp"]
	client, err := f.connection(s, binding)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	component := &mysqlComponent{
		baseComponent: baseComponent{
			name: s.Name,
		},
		client:      client,
		credentials: s.Credentials,
		binding:     binding,
	}

	return component, nil
}

func (f mysqlFactory) connection(settings *mysqlSettings, binding containerBinding) (*sqlx.DB, error) {
	err := f.setup(settings, binding)
	if err != nil {
		return nil, fmt.Errorf("can not prepare database: %w", err)
	}

	dsn := url.URL{
		User: url.UserPassword(settings.Credentials.UserName, settings.Credentials.UserPassword),
		Host: fmt.Sprintf("tcp(%s:%v)", binding.host, binding.port),
		Path: settings.Credentials.DatabaseName,
	}

	qry := dsn.Query()
	qry.Set("multiStatements", "true")
	qry.Set("parseTime", "true")
	qry.Set("charset", "utf8mb4")
	dsn.RawQuery = qry.Encode()

	client, err := sqlx.Open("mysql", dsn.String()[2:])
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return client, nil
}

func (f mysqlFactory) setup(settings *mysqlSettings, binding containerBinding) error {
	dsn := url.URL{
		User: url.UserPassword("root", settings.Credentials.RootPassword),
		Host: fmt.Sprintf("tcp(%s:%v)", binding.host, binding.port),
		Path: "/",
	}

	client, err := sql.Open("mysql", dsn.String()[2:])
	if err != nil {
		return fmt.Errorf("can not create root client: %w", err)
	}

	err = client.Ping()
	if err != nil {
		return fmt.Errorf("unable to connect via root: %w", err)
	}

	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", settings.Credentials.DatabaseName)

	_, err = client.Exec(createDB)
	if err != nil {
		return fmt.Errorf("can not create database: %w", err)
	}

	createUser := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", settings.Credentials.UserName, settings.Credentials.UserPassword)

	_, err = client.Exec(createUser)
	if err != nil {
		return fmt.Errorf("can not create test user: %w", err)
	}

	grantPermissions := fmt.Sprintf("GRANT ALL ON `%s`.* TO '%s'@'%%';", settings.Credentials.DatabaseName, settings.Credentials.UserName)

	_, err = client.Exec(grantPermissions)
	if err != nil {
		return fmt.Errorf("can not grant permissions for test user: %w", err)
	}

	return nil
}

func (f mysqlFactory) dropDatabase(settings interface{}) ComponentShutdownCallback {
	return func(container *container) func() error {
		return func() error {
			s := settings.(*mysqlSettings)
			binding := container.bindings["3306/tcp"]

			client, err := f.connection(s, binding)
			if err != nil {
				return fmt.Errorf("can not connect to database: %w", err)
			}

			dropDatabase := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", s.Credentials.DatabaseName)

			_, err = client.Exec(dropDatabase)
			if err != nil {
				return fmt.Errorf("can not drop database: %w", err)
			}

			return nil
		}
	}
}
