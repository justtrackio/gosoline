package env

import (
	"database/sql"
	"fmt"
	"net/url"
	"sync"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentMySql] = new(mysqlFactory)
}

const componentMySql = "mysql"

type mysqlCredentials struct {
	DatabaseName string `cfg:"database_name" default:"gosoline"`
	UserName     string `cfg:"user_name"     default:"gosoline"`
	UserPassword string `cfg:"user_password" default:"gosoline"`
	RootPassword string `cfg:"root_password" default:"gosoline"`
}

type mysqlSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	ContainerBindingSettings
	Credentials      mysqlCredentials `cfg:"credentials"`
	ToxiproxyEnabled bool             `cfg:"toxiproxy_enabled" default:"false"`
}

type mysqlFactory struct {
	lck              sync.Mutex
	connections      map[string]*sqlx.DB
	toxiproxyFactory toxiproxyFactory
}

func (f *mysqlFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("db") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentMySql) {
		return nil
	}

	if has, err := manager.HasType(componentMySql); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	components, err := config.GetStringMap("db")
	if err != nil {
		return fmt.Errorf("can not get db components: %w", err)
	}

	for name := range components {
		driver, err := config.Get(fmt.Sprintf("db.%s.driver", name))
		if err != nil {
			return fmt.Errorf("can not get driver for component %s: %w", name, err)
		}

		if driver != componentMySql {
			continue
		}

		settings := &mysqlSettings{}
		if err := UnmarshalSettings(config, settings, componentMySql, "default"); err != nil {
			return fmt.Errorf("can not unmarshal mysql settings for component %s: %w", name, err)
		}
		settings.Type = componentMySql
		settings.Name = name

		if err := manager.Add(settings); err != nil {
			return fmt.Errorf("can not add default mysql component: %w", err)
		}
	}

	return nil
}

func (f *mysqlFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &mysqlSettings{}
}

func (f *mysqlFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*mysqlSettings)

	descriptions := ComponentContainerDescriptions{
		"main": {
			ContainerConfig:  f.configureContainer(settings),
			HealthCheck:      f.healthCheck(settings),
			ShutdownCallback: f.dropDatabase(settings),
		},
	}

	if s.ToxiproxyEnabled {
		descriptions["toxiproxy"] = f.toxiproxyFactory.describeContainer()
	}

	return descriptions
}

func (f *mysqlFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*mysqlSettings)

	env := map[string]string{
		"MYSQL_DATABASE":      s.Credentials.DatabaseName,
		"MYSQL_USER":          s.Credentials.UserName,
		"MYSQL_PASSWORD":      s.Credentials.UserPassword,
		"MYSQL_ROOT_PASSWORD": s.Credentials.RootPassword,
		"MYSQL_ROOT_HOST":     "%",
	}

	if len(s.Tmpfs) == 0 {
		s.Tmpfs = append(s.Tmpfs, TmpfsSettings{
			Path: "/var/lib/mysql",
		})
	}

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		Tmpfs:      s.Tmpfs,
		Env:        env,
		Cmd:        []string{"--sql_mode=NO_ENGINE_SUBSTITUTION", "--log-bin-trust-function-creators=TRUE", "--max_connections=1000"},
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 3306,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
	}
}

func (f *mysqlFactory) healthCheck(settings any) ComponentHealthCheck {
	return func(container *Container) error {
		s := settings.(*mysqlSettings)
		binding := container.bindings["main"]
		client, err := f.connection(s, binding)
		if err != nil {
			return fmt.Errorf("can not create client: %w", err)
		}

		return client.Ping()
	}
}

func (f *mysqlFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*mysqlSettings)

	var err error
	var con *sqlx.DB
	var proxy *toxiproxy.Proxy

	mysqlBinding := containers["main"].bindings["main"]

	if s.ToxiproxyEnabled {
		toxiproxyClient := f.toxiproxyFactory.client(containers["toxiproxy"])

		if proxy, err = toxiproxyClient.CreateProxy("ddb", ":56248", mysqlBinding.getAddress()); err != nil {
			return nil, fmt.Errorf("can not create toxiproxy proxy for ddb component: %w", err)
		}

		mysqlBinding = containers["toxiproxy"].bindings["main"]
	}

	if con, err = f.connection(s, mysqlBinding); err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	component := &mysqlComponent{
		baseComponent: baseComponent{
			name: s.Name,
		},
		client:      con,
		credentials: s.Credentials,
		binding:     mysqlBinding,
		toxiproxy:   proxy,
	}

	return component, nil
}

func (f *mysqlFactory) connection(settings *mysqlSettings, binding ContainerBinding) (*sqlx.DB, error) {
	dsn := url.URL{
		User: url.UserPassword(settings.Credentials.UserName, settings.Credentials.UserPassword),
		Host: fmt.Sprintf("tcp(%s:%v)", binding.host, binding.port),
		Path: settings.Credentials.DatabaseName,
	}

	f.lck.Lock()
	defer f.lck.Unlock()

	if f.connections == nil {
		f.connections = make(map[string]*sqlx.DB)
	}

	if _, ok := f.connections[dsn.String()]; !ok {
		err := f.setup(settings, binding)
		if err != nil {
			return nil, fmt.Errorf("can not prepare database: %w", err)
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

		f.connections[dsn.String()] = client
	}

	return f.connections[dsn.String()], nil
}

func (f *mysqlFactory) setup(settings *mysqlSettings, binding ContainerBinding) error {
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

	err = client.Close()
	if err != nil {
		return fmt.Errorf("can not close connection: %w", err)
	}

	return nil
}

func (f *mysqlFactory) dropDatabase(settings any) ComponentShutdownCallback {
	return func(container *Container) func() error {
		return func() error {
			s := settings.(*mysqlSettings)
			binding := container.bindings["main"]

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
