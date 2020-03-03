package test

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"net/http"
)

type elasticsearchSettings struct {
	*mockSettings
	Port    int    `cfg:"port"`
	Version string `cfg:"version"`
}

type elasticsearchComponent struct {
	name     string
	settings *elasticsearchSettings
	runner   *dockerRunner
}

func (e *elasticsearchComponent) Boot(config cfg.Config, runner *dockerRunner, settings *mockSettings, name string) {
	e.name = name
	e.runner = runner
	e.settings = &elasticsearchSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, e.settings)
}

func (e *elasticsearchComponent) Start() {
	containerName := fmt.Sprintf("gosoline_test_elasticsearch_%s", e.name)

	e.runner.Run(containerName, containerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        e.settings.Version,
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: portBinding{
			"9200/tcp": fmt.Sprint(e.settings.Port),
		},
		HealthCheck: func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s:%d/_cluster/health", e.settings.Host, e.settings.Port))

			if err != nil {
				return err
			}

			// elastic might not have completed its boot process yet
			if resp.StatusCode > 200 {
				return errors.New("not yet healthy")
			}

			return nil
		},
		PrintLogs: e.settings.Debug,
	})
}
