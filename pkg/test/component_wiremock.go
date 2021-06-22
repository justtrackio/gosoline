package test

import (
	"bytes"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

type wiremockSettings struct {
	*mockSettings
	Port  int    `cfg:"port" default:"0"`
	Mocks string `cfg:"mocks"`
}

type wiremockComponent struct {
	mockComponentBase
	settings *wiremockSettings
}

func (w *wiremockComponent) Boot(config cfg.Config, _ log.Logger, runner *dockerRunnerLegacy, settings *mockSettings, name string) {
	w.name = name
	w.runner = runner
	w.settings = &wiremockSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, w.settings)
}

func (w *wiremockComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_wiremock_%s", w.name)

	err := w.runner.Run(containerName, &containerConfigLegacy{
		Repository: "rodolpheche/wiremock",
		Tag:        "2.26.3-alpine",
		PortBindings: portBindingLegacy{
			"8080/tcp": fmt.Sprint(w.settings.Port),
		},
		PortMappings: portMappingLegacy{
			"8080/tcp": &w.settings.Port,
		},
		HostMapping: hostMappingLegacy{
			dialPort: &w.settings.Port,
			setHost:  &w.settings.Host,
		},
		HealthCheck: func() error {
			url := fmt.Sprintf("%s/", w.getUrl())

			resp, err := http.Get(url)

			if err == nil && resp.StatusCode >= 399 {
				err = fmt.Errorf("wiremock did return status '%s'", resp.Status)
			}

			return err
		},
		PrintLogs:   w.settings.Debug,
		ExpireAfter: w.settings.ExpireAfter,
	})

	if err != nil {
		return err
	}

	jsonStr, err := ioutil.ReadFile(w.settings.Mocks)

	if err != nil {
		filename := w.settings.Mocks

		absolutePath, err := filepath.Abs(filename)
		if err == nil {
			filename = absolutePath
		}

		return fmt.Errorf("could not read http mock configuration '%s': %w", filename, err)
	}

	url := w.getUrl()
	resp, err := http.Post(url+"/mappings/import", "application/json", bytes.NewBuffer(jsonStr))

	if err != nil {
		return fmt.Errorf("could not send stubs to wiremock: %w", err)
	}

	if resp.StatusCode > 399 {
		return fmt.Errorf("could not import mocks")
	}

	return nil
}

func (w *wiremockComponent) getUrl() string {
	return fmt.Sprintf("http://%s:%d/__admin", w.settings.Host, w.settings.Port)
}
