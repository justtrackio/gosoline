package env

import (
	"fmt"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

type toxiproxyFactory struct{}

func (f *toxiproxyFactory) describeContainer() *ComponentContainerDescription {
	return &ComponentContainerDescription{
		ContainerConfig: &ContainerConfig{
			Repository: "ghcr.io/shopify/toxiproxy",
			Tag:        "2.9.0",
			PortBindings: PortBindings{
				"admin": {ContainerPort: 8474, HostPort: 0, Protocol: "tcp"},
				"main":  {ContainerPort: 56248, HostPort: 0, Protocol: "tcp"},
			},
			ExposedPorts: []string{"56248"},
		},
		HealthCheck: func(container *Container) error {
			binding := container.bindings["admin"]
			address := fmt.Sprintf("%s:%s", binding.host, binding.port)
			client := toxiproxy.NewClient(address)

			_, err := client.Proxies()

			return err
		},
	}
}

func (f *toxiproxyFactory) client(container *Container) *toxiproxy.Client {
	binding := container.bindings["admin"]
	address := fmt.Sprintf("%s:%s", binding.host, binding.port)
	client := toxiproxy.NewClient(address)

	return client
}
