package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/mon"
)

type wiremockComponent struct {
	baseComponent
	logger  mon.Logger
	binding containerBinding
	client  http.Client
}

func (c *wiremockComponent) AppOptions() []application.Option {
	return []application.Option{}
}

func (c *wiremockComponent) Address() string {
	return fmt.Sprintf("http://%s:%s", c.binding.host, c.binding.port)
}

func (c *wiremockComponent) Client() http.Client {
	return c.client
}
