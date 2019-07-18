package test

import (
	"log"
	"net/http"
)

type elasticsearchConfig struct {
	Version string `mapstructure:"version"`
}

func runElasticsearch(name string, config configInput) {
	wait.Add(1)
	go doRunElasticsearch(name, config)
}

func doRunElasticsearch(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "elasticsearch")

	config := &elasticsearchConfig{}
	unmarshalConfig(configMap, config)

	runContainer("gosoline_test_elasticsearch", ContainerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        config.Version,
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: PortBinding{
			"9200/tcp": "9222",
		},
		HealthCheck: func() error {
			_, err := http.Get("http://localhost:9222/_cluster/health")
			return err
		},
	})
}
