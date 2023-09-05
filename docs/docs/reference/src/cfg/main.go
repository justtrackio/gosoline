package main

import (
	"fmt"
	"os"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type StructExample struct {
	Host      string        `cfg:"host" default:"exampl.com"` // "cfg" defines the yaml key and "default" defines the default is nothing is configured
	Port      int           `cfg:"port" default:"80"`
	Timeout   time.Duration `cfg:"timeout" default:"1s"`
	Threshold int           `cfg:"threshold" default:"10"`
}

func main() {
	// create
	config := cfg.New()

	// there are multiple options to add values
	options := []cfg.Option{
		// from a yaml file
		cfg.WithConfigFile("config.dist.yml", "yml"),
		// from a map
		cfg.WithConfigMap(map[string]interface{}{
			"enabled": true,
		}),
		// from a single setting
		cfg.WithConfigSetting("foo", "bar"),

		// will replace "." and "-" with "_" for env key loading
		cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
	}

	// apply the options
	if err := config.Option(options...); err != nil {
		panic(err)
	}

	// access config via setters
	fmt.Printf("got port: %d\n", config.GetInt("port"))
	fmt.Printf("got host: %s\n", config.GetString("host"))
	fmt.Printf("is enabled: %t\n", config.GetBool("enabled"))
	fmt.Printf("foo is: %s\n", config.GetString("foo"))

	// one can provide a default parameter which is used if key is not set
	fmt.Printf("missing is not set %s\n", config.GetString("missing", "but has a default"))

	// you can also unmarshal a map from yaml into a struct
	settings := &StructExample{}
	config.UnmarshalKey("struct_example", settings)

	fmt.Printf("%+v\n", settings)

	// environment variables will overwrite config values
	os.Setenv("PORT", "88")
	fmt.Printf("got port: %d\n", config.GetInt("port"))

	os.Setenv("STRUCT_EXAMPLE_TIMEOUT", "7s")
	config.UnmarshalKey("struct_example", settings)
	fmt.Printf("%+v\n", settings)
}
