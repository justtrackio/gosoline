package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/log"
)

type wiremockComponent struct {
	baseComponent
	logger  log.Logger
	binding containerBinding
	client  http.Client
}

func (c *wiremockComponent) CfgOptions() []cfg.Option {
	return []cfg.Option{}
}

func (c *wiremockComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
}

func (c *wiremockComponent) Client() http.Client {
	return c.client
}
