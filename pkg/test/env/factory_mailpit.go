package env

import (
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env/mailpit"
)

func init() {
	componentFactories[componentMailpit] = &mailpitFactory{}
}

const componentMailpit = "mailpit"

type mailpitSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Port    int `cfg:"port" default:"0"`
	WebPort int `cfg:"web_port" default:"0"`
}

type mailpitFactory struct{}

func (m mailpitFactory) Detect(config cfg.Config, manager *ComponentsConfigManager) error {
	if !config.IsSet("email") {
		return nil
	}

	if !manager.ShouldAutoDetect(componentMailpit) {
		return nil
	}

	if has, err := manager.HasType(componentMailpit); err != nil {
		return fmt.Errorf("failed to check if component exists: %w", err)
	} else if has {
		return nil
	}

	settings := &mailpitSettings{}
	if err := UnmarshalSettings(config, settings, componentMailpit, "default"); err != nil {
		return fmt.Errorf("can not unmarshal mailpit settings: %w", err)
	}
	settings.Type = componentMailpit

	if err := manager.Add(settings); err != nil {
		return fmt.Errorf("can not add default mailpit component: %w", err)
	}

	return nil
}

func (m mailpitFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &mailpitSettings{}
}

func (m mailpitFactory) DescribeContainers(settings any) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig:  m.configureContainer(settings.(*mailpitSettings)),
			healthCheck:      m.healthCheck(),
			shutdownCallback: nil,
		},
	}
}

func (m mailpitFactory) Component(_ cfg.Config, logger log.Logger, container map[string]*container, _ any) (Component, error) {
	main := container["main"]

	client := mailpit.NewClientWithInterfaces(http.Client{}, mailpit.Config{
		Server:   m.addressWeb(main),
		Protocol: "http",
	})

	return &mailpitComponent{
		baseComponent: baseComponent{},
		logger:        logger,
		client:        client,
		addressSmtp:   m.addressSmtp(main),
		addressWeb:    m.addressWeb(main),
	}, nil
}

func (m mailpitFactory) configureContainer(settings *mailpitSettings) *containerConfig {
	return &containerConfig{
		Auth:       settings.Image.Auth,
		Repository: settings.Image.Repository,
		Tag:        settings.Image.Tag,
		PortBindings: portBindings{
			"1025/tcp": settings.Port,
			"8025/tcp": settings.WebPort,
		},
		ExpireAfter: settings.ExpireAfter,
	}
}

func (m mailpitFactory) addressSmtp(c *container) string {
	binding := c.bindings["1025/tcp"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (m mailpitFactory) addressWeb(c *container) string {
	binding := c.bindings["8025/tcp"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)

	return address
}

func (m mailpitFactory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		url := fmt.Sprintf("http://%s/livez", m.addressWeb(container))

		var err error
		var resp *http.Response

		if resp, err = http.Get(url); err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
		}

		return nil
	}
}
