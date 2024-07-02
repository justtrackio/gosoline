package env

import (
	"fmt"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

type toxiproxyFactory struct{}

func (f *toxiproxyFactory) describeContainer(expireAfter time.Duration) *componentContainerDescription {
	return &componentContainerDescription{
		containerConfig: &containerConfig{
			Repository: "ghcr.io/shopify/toxiproxy",
			Tag:        "2.9.0",
			PortBindings: portBindings{
				"8474/tcp":  0,
				"56248/tcp": 0,
			},
			ExposedPorts: []string{"56248"},
			ExpireAfter:  expireAfter,
		},
		healthCheck: func(container *container) error {
			binding := container.bindings["8474/tcp"]
			address := fmt.Sprintf("%s:%s", binding.host, binding.port)
			client := toxiproxy.NewClient(address)

			_, err := client.Proxies()

			return err
		},
	}
}

func (f *toxiproxyFactory) client(container *container) *toxiproxy.Client {
	binding := container.bindings["8474/tcp"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)
	client := toxiproxy.NewClient(address)

	return client
}
