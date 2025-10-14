package env

import (
	"bytes"
	"fmt"
	"io"
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
	Mocks []string `cfg:"mocks"`
	Port  int      `cfg:"port" default:"0"`
}

type wiremockFactory struct{}

func (f *wiremockFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *wiremockFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &wiremockSettings{}
}

func (f *wiremockFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings),
			HealthCheck:     f.healthCheck(),
		},
	}
}

func (f *wiremockFactory) configureContainer(settings any) *ContainerConfig {
	s := settings.(*wiremockSettings)

	return &ContainerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
		PortBindings: PortBindings{
			"main": {
				ContainerPort: 8080,
				HostPort:      s.Port,
				Protocol:      "tcp",
			},
		},
		Cmd: []string{"--local-response-templating"},
	}
}

func (f *wiremockFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		binding := container.bindings["main"]
		url := fmt.Sprintf("%s/", f.getUrl(binding))

		resp, err := http.Get(url)

		if err == nil && resp.StatusCode >= 399 {
			err = fmt.Errorf("wiremock did return status '%s'", resp.Status)
		}

		return err
	}
}

func (f *wiremockFactory) Component(_ cfg.Config, logger log.Logger, containers map[string]*Container, settings any) (Component, error) {
	component := &wiremockComponent{
		logger:  logger,
		binding: containers["main"].bindings["main"],
	}

	s := settings.(*wiremockSettings)
	url := f.getUrl(containers["main"].bindings["main"])

	for _, mock := range s.Mocks {
		if err := f.importMocks(url, mock); err != nil {
			return nil, fmt.Errorf("could not import mocks from file %q: %w", mock, err)
		}
	}

	return component, nil
}

func (f *wiremockFactory) importMocks(url string, mockFile string) error {
	var err error
	var jsonBytes, body []byte
	var absolutePath string
	var resp *http.Response

	if jsonBytes, err = os.ReadFile(mockFile); err != nil {
		filename := mockFile

		if absolutePath, err = filepath.Abs(filename); err == nil {
			filename = absolutePath
		}

		return fmt.Errorf("could not read http mock configuration '%s': %w", filename, err)
	}

	if resp, err = http.Post(url+"/mappings/import", "application/json", bytes.NewBuffer(jsonBytes)); err != nil {
		return fmt.Errorf("could not send stubs to wiremock: %w", err)
	}

	if resp.StatusCode < 400 {
		return nil
	}

	if body, err = io.ReadAll(resp.Body); err != nil {
		return fmt.Errorf("could not read wiremock response body: %w", err)
	}

	return fmt.Errorf("could not import mocks: %s", body)
}

func (f *wiremockFactory) getUrl(binding ContainerBinding) string {
	return fmt.Sprintf("http://%s:%s/__admin", binding.host, binding.port)
}
