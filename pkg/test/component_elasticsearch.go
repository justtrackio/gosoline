package test

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"log"
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
}

func (m *elasticsearchComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	m.name = name
	m.settings = &elasticsearchSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, m.settings)
}

func (m *elasticsearchComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", m.name, "elasticsearch")

	containerName := fmt.Sprintf("gosoline_test_elasticsearch_%s", m.name)

	runner.Run(containerName, containerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        m.settings.Version,
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: portBinding{
			"9200/tcp": fmt.Sprint(m.settings.Port),
		},
		HealthCheck: func() error {
			resp, err := http.Get(fmt.Sprintf("http://%s:%d/_cluster/health", m.settings.Host, m.settings.Port))

			if err != nil {
				return err
			}

			// elastic might not have completed its boot process yet
			if resp.StatusCode > 200 {
				return errors.New("not yet healthy")
			}

			return nil
		},
		PrintLogs: m.settings.Debug,
	})
}

func (m *elasticsearchComponent) ProvideClient(string) interface{} {
	return nil // no client needed for elasticsearch
}
