package env

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

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
	ContainerBindingSettings
	Mocks                []string `cfg:"mocks"`
	UseExternalContainer bool     `cfg:"use_external_container" default:"false"`
}

type wiremockFactory struct{}

func (f *wiremockFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *wiremockFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &wiremockSettings{}
}

func (f *wiremockFactory) DescribeContainers(settings any) componentContainerDescriptions {
	return componentContainerDescriptions{
		"main": {
			containerConfig: f.configureContainer(settings),
			healthCheck:     f.healthCheck(),
		},
	}
}

func (f *wiremockFactory) configureContainer(settings any) *containerConfig {
	s := settings.(*wiremockSettings)

	if s.UseExternalContainer {
		return &containerConfig{
			UseExternalContainer: true,
			ContainerBindings: containerBindings{
				"8080/tcp": containerBinding{
					host: s.Host,
					port: strconv.Itoa(s.Port),
				},
			},
		}
	}

	// ensure to use a free port for the new container
	s.Port = 0

	return &containerConfig{
		Auth:       s.Image.Auth,
		Repository: s.Image.Repository,
		Tag:        s.Image.Tag,
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

func (f *wiremockFactory) Component(_ cfg.Config, logger log.Logger, containers map[string]*container, settings any) (Component, error) {
	component := &wiremockComponent{
		logger:  logger,
		binding: containers["main"].bindings["8080/tcp"],
	}

	s := settings.(*wiremockSettings)
	url := f.getUrl(containers["main"].bindings["8080/tcp"])

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

func (f *wiremockFactory) getUrl(binding containerBinding) string {
	return fmt.Sprintf("http://%s:%s/__admin", binding.host, binding.port)
}
