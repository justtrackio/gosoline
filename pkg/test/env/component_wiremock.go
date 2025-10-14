package env

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

type wiremockComponent struct {
	baseComponent
	logger  log.Logger
	binding ContainerBinding
	client  http.Client
}

func (c *wiremockComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{}
}

func (c *wiremockComponent) GetHost() string {
	return c.binding.host
}

func (c *wiremockComponent) GetPort() string {
	return c.binding.port
}

func (c *wiremockComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
}

func (c *wiremockComponent) Client() http.Client {
	return c.client
}
