package env

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	componentFactories[componentMmock] = new(mmockFactory)
}

const componentMmock = "mmock"

type mmockSettings struct {
	ComponentBaseSettings
	ComponentContainerSettings
	Mocks     []string `cfg:"mocks"`
	Port      int      `cfg:"port" default:"0"`
	AdminPort int      `cfg:"admin_port" default:"0"`
}

type mmockFactory struct{}

func (f *mmockFactory) Detect(_ cfg.Config, _ *ComponentsConfigManager) error {
	return nil
}

func (f *mmockFactory) GetSettingsSchema() ComponentBaseSettingsAware {
	return &mmockSettings{}
}

func (f *mmockFactory) DescribeContainers(settings any) ComponentContainerDescriptions {
	s := settings.(*mmockSettings)
	tmpfs := s.Tmpfs
	if len(tmpfs) == 0 {
		tmpfs = []TmpfsSettings{{Path: "/config", Size: "16m"}}
	}

	return ComponentContainerDescriptions{
		"main": {
			ContainerConfig: &ContainerConfig{
				Auth:       s.Image.Auth,
				Repository: s.Image.Repository,
				Tag:        s.Image.Tag,
				Tmpfs:      tmpfs,
				Env:        map[string]string{"LOG_LEVEL": "ERROR"},
				PortBindings: PortBindings{
					"main":  {ContainerPort: 8083, HostPort: s.Port, Protocol: "tcp"},
					"admin": {ContainerPort: 8082, HostPort: s.AdminPort, Protocol: "tcp"},
				},
				Cmd: []string{"-server-ip", "0.0.0.0", "-console-ip", "0.0.0.0", "-server-statistics=false", "-request-storage-capacity", "1000"},
			},
			HealthCheck: func(container *Container) error {
				component := &MmockComponent{adminBinding: container.bindings["admin"]}
				_, err := component.request(context.Background(), http.MethodGet, "/api/mapping", nil)

				return err
			},
		},
	}
}

func (f *mmockFactory) Component(_ cfg.Config, _ log.Logger, containers map[string]*Container, settings any) (Component, error) {
	s := settings.(*mmockSettings)
	component := &MmockComponent{
		baseComponent: baseComponent{name: s.Name},
		binding:       containers["main"].bindings["main"],
		adminBinding:  containers["main"].bindings["admin"],
	}
	for i, filename := range s.Mocks {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("could not read mmock fixture %q: %w", filename, err)
		}
		name := fmt.Sprintf("%d-%s", i, filepath.Base(filename))
		if _, err = component.request(context.Background(), http.MethodPost, "/api/mapping/"+url.PathEscape(name), bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("could not load mmock fixture %q: %w", filename, err)
		}
	}

	return component, nil
}
