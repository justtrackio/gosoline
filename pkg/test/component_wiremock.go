package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type wiremockSettings struct {
	*mockSettings
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

func (w *wiremockComponent) getContainerConfig() *containerConfigLegacy {
	return &containerConfigLegacy{
		Repository: "wiremock/wiremock",
		// alpine version doesn't run on arm based chips that support x86/x64 emulation, main does have an arm version but is not a specific version
		Tag: "2.32.0",
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
	}
}

func (w *wiremockComponent) PullContainerImage() error {
	containerName := fmt.Sprintf("gosoline_test_wiremock_%s", w.name)
	containerConfig := w.getContainerConfig()

	return w.runner.PullContainerImage(containerName, containerConfig)
}

func (w *wiremockComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_wiremock_%s", w.name)
	containerConfig := w.getContainerConfig()

	err := w.runner.Run(containerName, containerConfig)
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
