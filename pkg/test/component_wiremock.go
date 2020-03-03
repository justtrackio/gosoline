package test

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"io/ioutil"
	"log"
	"net/http"
)

type wiremockSettings struct {
	*mockSettings
	Port  uint   `cfg:"port"`
	Mocks string `cfg:"mocks"`
}

type wiremockComponent struct {
	name     string
	db       *sql.DB
	settings *wiremockSettings
}

func (w *wiremockComponent) Boot(name string, config cfg.Config, settings *mockSettings) {
	w.name = name
	w.settings = &wiremockSettings{
		mockSettings: settings,
	}
	key := fmt.Sprintf("mocks.%s", name)
	config.UnmarshalKey(key, w.settings)
}

func (w *wiremockComponent) Run(runner *dockerRunner) {
	defer log.Printf("%s component of type %s is ready", w.name, "wiremock")

	url := fmt.Sprintf("http://%s:%d/__admin", w.settings.Host, w.settings.Port)

	containerName := fmt.Sprintf("gosoline_test_wiremock_%s", w.name)

	runner.Run(containerName, containerConfig{
		Repository: "rodolpheche/wiremock",
		Tag:        "latest",
		PortBindings: portBinding{
			"8080/tcp": fmt.Sprint(w.settings.Port),
		},
		HealthCheck: func() error {
			_, err := http.Get(url)

			return err
		},
	})

	jsonStr, err := ioutil.ReadFile(w.settings.Mocks)

	if err != nil {
		panic(fmt.Errorf("could not read http mock configuration : %w", err))
	}

	_, err = http.Post(url+"/mappings/import", "application/json", bytes.NewBuffer(jsonStr))

	if err != nil {
		panic(fmt.Errorf("could not send stubs to wiremock : %w", err))
	}
}

func (w *wiremockComponent) ProvideClient(string) interface{} {
	return nil // no client needed for wiremock
}
