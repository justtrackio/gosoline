package env

import (
	"time"

	"github.com/ory/dockertest/v3/docker"
)

type AutoDetectSettings struct {
	Enabled        bool     `cfg:"enabled"         default:"true"`
	SkipComponents []string `cfg:"skip_components"`
}

type ComponentBaseSettings struct {
	Name string `cfg:"name" default:"default"`
	Type string `cfg:"type"                   validate:"required"`
}

func (c *ComponentBaseSettings) GetName() string {
	return c.Name
}

func (c *ComponentBaseSettings) GetType() string {
	return c.Type
}

func (c *ComponentBaseSettings) SetName(name string) {
	c.Name = name
}

func (c *ComponentBaseSettings) SetType(typ string) {
	c.Type = typ
}

type ContainerImageSettings struct {
	Repository string `cfg:"repository"`
	Tag        string `cfg:"tag"`
}

type ContainerBindingSettings struct {
	Host string `cfg:"host" default:"127.0.0.1"`
	Port int    `cfg:"port" default:"0"`
}

type ComponentContainerSettings struct {
	Image       ContainerImageSettings `cfg:"image"`
	ExpireAfter time.Duration          `cfg:"expire_after" default:"60s"`
	Tmpfs       []TmpfsSettings        `cfg:"tmpfs"`
}

type TmpfsSettings struct {
	Path string `cfg:"path"`
	Size string `cfg:"size"`
	Mode string `cfg:"mode"`
}

type healthCheckSettings struct {
	InitialInterval time.Duration `cfg:"initial_interval" default:"1s"`
	MaxInterval     time.Duration `cfg:"max_interval"     default:"3s"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"1m"`
}

type authSettings struct {
	Username      string `cfg:"username"       default:""`
	Password      string `cfg:"password"       default:""`
	Email         string `cfg:"email"          default:""`
	ServerAddress string `cfg:"server_address" default:""`
}

func (a authSettings) GetAuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{
		Username:      a.Username,
		Password:      a.Password,
		Email:         a.Email,
		ServerAddress: a.ServerAddress,
	}
}

type containerRunnerSettings struct {
	Endpoint    string              `cfg:"endpoint"`
	NamePrefix  string              `cfg:"name_prefix"  default:"goso"`
	HealthCheck healthCheckSettings `cfg:"health_check"`
	ExpireAfter time.Duration       `cfg:"expire_after"`
	Auth        authSettings        `cfg:"auth"`
}

type ddbSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port             int  `cfg:"port"              default:"0"`
	ToxiproxyEnabled bool `cfg:"toxiproxy_enabled" default:"false"`
}

type localstackSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port     int      `cfg:"port"     default:"0"`
	Region   string   `cfg:"region"   default:"eu-central-1"`
	Services []string `cfg:"services"`
}

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
	Credentials          mysqlCredentials `cfg:"credentials"`
	ToxiproxyEnabled     bool             `cfg:"toxiproxy_enabled"      default:"false"`
	UseExternalContainer bool             `cfg:"use_external_container" default:"false"`
}

type redisSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type s3Settings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port int `cfg:"port" default:"0"`
}

type streamInputSettings struct {
	ComponentBaseSettings
	InMemoryOverride bool `cfg:"in_memory_override" default:"true"`
}

type streamOutputSettings struct {
	ComponentBaseSettings
}

type wiremockSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Mocks []string `cfg:"mocks"`
	Port  int      `cfg:"port"  default:"0"`
}
