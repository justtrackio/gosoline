package test

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
)

type wiremockConfig struct {
	Mocks string `mapstructure:"mocks"`
}

func runWiremock(name string, config configInput) {
	wait.Add(1)
	go doRunWiremock(name, config)
}

func doRunWiremock(name string, configMap configInput) {
	defer wait.Done()
	defer log.Printf("%s component of type %s is ready", name, "wiremock")

	runContainer("gosoline_test_wiremock", ContainerConfig{
		Repository: "rodolpheche/wiremock",
		Tag:        "latest",
		PortBindings: PortBinding{
			"8080/tcp": "8888",
		},
		HealthCheck: func() error {
			_, err := http.Get("http://localhost:8888/__admin/")
			return err
		},
	})

	config := &wiremockConfig{}
	unmarshalConfig(configMap, config)

	jsonStr, err := ioutil.ReadFile(config.Mocks)

	if err != nil {
		logErr(err, "could not read http mock configuration")
	}

	_, err = http.Post("http://localhost:8888/__admin/mappings/import", "application/json", bytes.NewBuffer(jsonStr))

	if err != nil {
		logErr(err, "could not send stubs to wiremock")
	}
}
