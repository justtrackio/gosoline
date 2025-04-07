package env

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env/mailpit"
)

type mailpitComponent struct {
	baseComponent
	logger      log.Logger
	addressSmtp string
	addressWeb  string
	client      mailpit.Client
}

func (c *mailpitComponent) CfgOptions() []cfg.Option {
	opts := []cfg.Option{
		cfg.WithConfigSetting("email", map[string]any{
			"default": map[string]string{
				"type":         "smtp",
				"server":       c.addressSmtp,
				"from_address": "system@marketing-sandbox.info",
			},
		}),
	}

	return opts
}

func (c *mailpitComponent) Client() mailpit.Client {
	return c.client
}

func (c *mailpitComponent) Address() string {
	return c.addressWeb
}
