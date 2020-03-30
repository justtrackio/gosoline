package test

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/es"
	"github.com/applike/gosoline/pkg/mon"
	"net/http"
)

type elasticsearchSettings struct {
	*mockSettings
	Port    int    `cfg:"port" default:"0"`
	Version string `cfg:"version"`
}

type elasticsearchComponent struct {
	mockComponentBase
	settings *elasticsearchSettings
	clients  *simpleCache
}

func (e *elasticsearchComponent) Boot(config cfg.Config, logger mon.Logger, runner *dockerRunner, settings *mockSettings, name string) {
	e.logger = logger
	e.name = name
	e.runner = runner
	e.settings = &elasticsearchSettings{
		mockSettings: settings,
	}
	e.clients = &simpleCache{}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, e.settings)
}

func (e *elasticsearchComponent) Start() error {
	containerName := fmt.Sprintf("gosoline_test_elasticsearch_%s", e.name)

	return e.runner.Run(containerName, containerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        e.settings.Version,
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: portBinding{
			"9200/tcp": fmt.Sprint(e.settings.Port),
		},
		PortMappings: map[string]*int{
			"9200/tcp": &e.settings.Port,
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
		PrintLogs:   e.settings.Debug,
		ExpireAfter: e.settings.ExpireAfter,
	})
}

func (e *elasticsearchComponent) provideElasticsearchV6Client(clientType string) *es.ClientV6 {
	return e.clients.New(e.name, func() interface{} {
		url := fmt.Sprintf("http://%s:%d", e.settings.Host, e.settings.Port)
		client := es.NewSimpleClientV6(e.logger, url, clientType)

		return client
	}).(*es.ClientV6)
}

func (e *elasticsearchComponent) provideElasticsearchV7Client(clientType string) *es.ClientV7 {
	return e.clients.New(e.name, func() interface{} {
		url := fmt.Sprintf("http://%s:%d", e.settings.Host, e.settings.Port)
		client := es.NewSimpleClient(e.logger, url, clientType)

		return client
	}).(*es.ClientV7)
}
