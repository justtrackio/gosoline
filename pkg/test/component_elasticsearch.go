package test

import (
	"fmt"
	"log"
	"net/http"
)

type elasticsearchConfig struct {
	Version string `mapstructure:"version"`
	Port    int    `mapstructure:"port"`
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

	containerName := fmt.Sprintf("gosoline_test_%s_elasticsearch", name)
	runContainer(containerName, ContainerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        config.Version,
		Env: []string{
			"discovery.type=single-node",
		},
		PortBindings: PortBinding{
			"9200/tcp": fmt.Sprint(config.Port),
		},
		HealthCheck: func() error {
			_, err := http.Get(fmt.Sprintf("http://localhost:%d/_cluster/health", config.Port))
			return err
		},
	})
}
