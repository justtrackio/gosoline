package env

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentMmock] = new(mmockFactory)
}

const componentMmock = "mmock"

type mmockFactory struct{}

type mmockSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	AdminPort int      `cfg:"admin_port" default:"0"`
	Mocks     []string `cfg:"mocks"`
	Port      int      `cfg:"port" default:"0"`
}

func (f *mmockFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*mmockSettings)
	adminAddress := f.getUrl(containers["main"].bindings["admin"])
	component := &MmockComponent{
		baseComponent: baseComponent{name: s.Name},
		address:       f.getUrl(containers["main"].bindings["main"]),
	}

	for i, filename := range s.Mocks {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("could not read mmock fixture %q: %w", filename, err)
		}

		name := fmt.Sprintf("%d-%s", i, filepath.Base(filename))
		if err = f.request(context.Background(), adminAddress, http.MethodPost, "/api/mapping/"+url.PathEscape(name), bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("could not load mmock fixture %q: %w", filename, err)
		}
	}

	return component, nil
}

func (f *mmockFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: f.configureContainer(settings.(*mmockSettings)),
			HealthCheck:     f.healthCheck(),
		},
	}
}

func (f *mmockFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *mmockFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &mmockSettings{}
}

func (f *mmockFactory) configureContainer(settings *mmockSettings) *ContainerConfig {
	tmpfs := settings.Tmpfs
	if len(tmpfs) == 0 {
		tmpfs = []TmpfsSettings{{Path: "/config", Size: "16m"}}
	}

	return &ContainerConfig{
		Auth:       settings.Image.Auth,
		Repository: settings.Image.Repository,
		Tag:        settings.Image.Tag,
		Tmpfs:      tmpfs,
		Env:        map[string]string{"LOG_LEVEL": "ERROR"},
		PortBindings: PortBindings{
			"admin": {ContainerPort: 8082, HostPort: settings.AdminPort, Protocol: "tcp"},
			"main":  {ContainerPort: 8083, HostPort: settings.Port, Protocol: "tcp"},
		},
		Cmd: []string{"-server-ip", "0.0.0.0", "-console-ip", "0.0.0.0", "-server-statistics=false", "-request-storage-capacity", "1000"},
	}
}

func (f *mmockFactory) getUrl(binding ContainerBinding) string {
	return "http://" + binding.getAddress()
}

func (f *mmockFactory) healthCheck() ComponentHealthCheck {
	return func(container *Container) error {
		address := f.getUrl(container.bindings["admin"])

		return f.request(context.Background(), address, http.MethodGet, "/api/mapping", nil)
	}
}

func (f *mmockFactory) request(ctx context.Context, address, method, path string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, method, address+path, body)
	if err != nil {
		return fmt.Errorf("could not create mmock request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not call mmock API: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err = errors.Join(err, resp.Body.Close()); err != nil {
		return fmt.Errorf("could not read mmock response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("mmock API %s %s returned %s: %s", method, path, resp.Status, data)
	}

	return nil
}
