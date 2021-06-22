package cloud

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	"strings"
)

//go:generate mockery -name DiscoveryClient
type DiscoveryClient interface {
	GetServiceInstances(application string) []string
	GetPipelineInstances(application string) map[string][]string
}

type discoveryClient struct {
	client    servicediscoveryiface.ServiceDiscoveryAPI
	logger    log.Logger
	namespace string
}

func GetDiscoveryClient(config cfg.Config, logger log.Logger) DiscoveryClient {
	endpoint := config.GetString("aws_serviceDiscovery_endpoint")
	client := GetServiceDiscoveryClient(logger, endpoint)

	return GetDiscoveryClientWithInterfaces(client, config, logger)
}

func GetDiscoveryClientWithInterfaces(client servicediscoveryiface.ServiceDiscoveryAPI, config cfg.Config, logger log.Logger) DiscoveryClient {
	namespace := config.GetString("aws_serviceDiscovery_namespace")

	return &discoveryClient{
		client:    client,
		logger:    logger,
		namespace: namespace,
	}
}

func (c *discoveryClient) GetServiceInstances(application string) []string {
	instances := make([]string, 0, 16)
	out, err := c.listServices()

	if err != nil {
		return instances
	}

	for _, svc := range out.Services {
		serviceName := *svc.Name

		if !strings.HasPrefix(serviceName, application) {
			continue
		}

		out, _ := c.client.ListInstances(&servicediscovery.ListInstancesInput{
			ServiceId: svc.Id,
		})

		for _, instance := range out.Instances {
			address := *instance.Attributes["AWS_INSTANCE_IPV4"] + ":" + *instance.Attributes["AWS_INSTANCE_PORT"]

			instances = append(instances, address)
		}
	}

	return instances
}

func (c *discoveryClient) GetPipelineInstances(application string) map[string][]string {
	out, err := c.listServices()

	instances := make(map[string][]string)
	instances["all"] = make([]string, 0)

	if err != nil {
		return instances
	}

	for _, svc := range out.Services {
		serviceName := *svc.Name

		if !strings.HasPrefix(serviceName, application) {
			continue
		}

		eventType := serviceName[len(application)+1:]
		instances[eventType] = make([]string, 0)

		out, _ := c.client.ListInstances(&servicediscovery.ListInstancesInput{
			ServiceId: svc.Id,
		})

		for _, instance := range out.Instances {
			address := *instance.Attributes["AWS_INSTANCE_IPV4"] + ":" + *instance.Attributes["AWS_INSTANCE_PORT"]

			instances["all"] = append(instances["all"], address)
			instances[eventType] = append(instances[eventType], address)
		}
	}

	return instances
}

func (c *discoveryClient) listServices() (*servicediscovery.ListServicesOutput, error) {
	filter := []*servicediscovery.ServiceFilter{
		{
			Name:      aws.String("NAMESPACE_ID"),
			Condition: aws.String(servicediscovery.FilterConditionEq),
			Values:    aws.StringSlice([]string{c.namespace}),
		},
	}

	out, err := c.client.ListServices(&servicediscovery.ListServicesInput{
		Filters: filter,
	})

	if err != nil {
		c.logger.Error("Could not get list from service discovery: %w", err)

		return nil, err
	}

	return out, nil
}
