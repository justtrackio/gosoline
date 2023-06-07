package env

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentWiremock] = new(wiremockFactory)
}

const componentWiremock = "wiremock"

type wiremockSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Mocks string `cfg:"mocks"`
	Port  int    `cfg:"port" default:"0"`
}

type wiremockFactory struct{}

func (f *wiremockFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *wiremockFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &wiremockSettings{}
}

func (f *wiremockFactory) DescribeContainers(settings interface{}) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}
}

func (f *wiremockFactory) configureContainer(settings interface{}) *containerConfig {
	s := settings.(*wiremockSettings)

	return &containerConfig{
		Repository: "wiremock/wiremock",
		// alpine version doesn't run on arm based chips that support x86/x64 emulation, main does have an arm version but is not a specific version
		Tag: "2.35.0",
		PortBindings: portBindings{
			"8080/tcp": s.Port,
		},
		ExpireAfter: s.ExpireAfter,
		Cmd:         []string{"--local-response-templating"},
	}
}

func (f *wiremockFactory) healthCheck() ComponentHealthCheck {
	return func(container *container) error {
		binding := container.bindings["8080/tcp"]
		url := fmt.Sprintf("%s/", f.getUrl(binding))

		resp, err := http.Get(url)

		if err == nil && resp.StatusCode >= 399 {
			err = fmt.Errorf("wiremock did return status '%s'", resp.Status)
		}

		return err
	}
}

func (f *wiremockFactory) Component(_ cfg.Config, logger log.Logger, containers map[string]*container, settings interface{}) (Component, error) {
	component := &wiremockComponent{
		logger:  logger,
		binding: containers["main"].bindings["8080/tcp"],
	}

	s := settings.(*wiremockSettings)
	jsonStr, err := os.ReadFile(s.Mocks)
	if err != nil {
		filename := s.Mocks

		absolutePath, err := filepath.Abs(filename)
		if err == nil {
			filename = absolutePath
		}

		return nil, fmt.Errorf("could not read http mock configuration '%s': %w", filename, err)
	}

	url := f.getUrl(containers["main"].bindings["8080/tcp"])
	resp, err := http.Post(url+"/mappings/import", "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, fmt.Errorf("could not send stubs to wiremock: %w", err)
	}

	if resp.StatusCode > 399 {
		return nil, fmt.Errorf("could not import mocks")
	}

	return component, nil
}

func (f *wiremockFactory) getUrl(binding containerBinding) string {
	return fmt.Sprintf("http://%s:%s/__admin", binding.host, binding.port)
}
