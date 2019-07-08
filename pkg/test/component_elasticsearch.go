package test

import (
	"log"
	"net/http"
)

func runElasticsearch(name string, config configMap) {
	wait.Add(1)
	go doRunElasticsearch(name, config)
}

func doRunElasticsearch(name string, config configMap) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "elasticsearch")

	version := configString(config, name, "version")

	runContainer("gosoline_test_elasticsearch", ContainerConfig{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        version,
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
