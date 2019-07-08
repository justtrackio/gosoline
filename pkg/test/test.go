package test

import (
	"errors"
	"fmt"
	"github.com/ory/dockertest"
	"log"
	"sync"
)

var err error
var wait sync.WaitGroup
var dockerPool *dockertest.Pool
var dockerResources []*dockertest.Resource

func init() {
	dockerPool, err = dockertest.NewPool("")
	dockerResources = make([]*dockertest.Resource, 0)

	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
}

func logErr(err error, msg string) {
	Shutdown()
	log.Println(msg)
	log.Fatal(err)
}

func Boot() {
	var ok bool
	var name string
	var config configMap

	configs := readConfig()

	for n, c := range configs {
		if name, ok = n.(string); !ok {
			logErr(errors.New("invalid type assert"), "name of the component should be string")
		}

		if config, ok = c.(configMap); !ok {
			logErr(errors.New("invalid type assert"), "type of the component config should be map")
		}

		bootComponent(name, config)
	}

	wait.Wait()

	log.Println("test environment up and running")
	fmt.Println()
}

func bootComponent(name string, config configMap) {
	component := configString(config, "main", "component")

	switch component {
	case "dynamodb":
		runDynamoDb(name, config)
	case "wiremock":
		runWiremock(name, config)
	case "elasticsearch":
		runElasticsearch(name, config)
	default:
		err := fmt.Errorf("unknown component '%s'", component)
		logErr(err, err.Error())
	}
}

func Shutdown() {
	for _, res := range dockerResources {
		if err := dockerPool.Purge(res); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}
}
