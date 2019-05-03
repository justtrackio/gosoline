package cloud

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/thoas/go-funk"
	"regexp"
	"strings"
)

//go:generate mockery -name ServiceClient
type ServiceClient interface {
	SetClient(client ecsiface.ECSAPI)
	Start(filter *FilterServicesInput, count int) ([]*ecs.Service, error)
	Stop(filter *FilterServicesInput) ([]*ecs.Service, error)
	ScaleServices(filter *FilterServicesInput, count int)
	ForceNewDeployment(filter *FilterServicesInput) error
	GetServices(filter *FilterServicesInput) ([]*ecs.Service, error)
	WaitUntilServiceIsStable(filter *FilterServicesInput)
	GetServiceList(filter *FilterServicesInput) []ServiceListing
	GetListingFromArn(arn *string) ServiceListing
}

type AwsServiceClient struct {
	client      ecsiface.ECSAPI
	logger      mon.Logger
	clusterName string
}

type ServiceListing struct {
	Arn         string
	Name        string
	Application string
	EventType   string
}

func GetServiceClient(client ecsiface.ECSAPI, logger mon.Logger, environment string) ServiceClient {
	return &AwsServiceClient{
		client:      client,
		logger:      logger,
		clusterName: fmt.Sprintf("mcoins-%v-analytics", environment),
	}
}

func GetServiceClientWithDefaultClient(logger mon.Logger, environment string) ServiceClient {
	client := GetEcsClient(logger)

	return GetServiceClient(client, logger, environment)
}

func (c *AwsServiceClient) SetClient(client ecsiface.ECSAPI) {
	c.client = client
}

type FilterServicesInput struct {
	Applications []string
	EventType    []string
}

type ScaleServicesInput struct {
	Filter FilterServicesInput `binding:"required"`
	Count  int                 `binding:"required"`
}

func (c *AwsServiceClient) Start(filter *FilterServicesInput, count int) ([]*ecs.Service, error) {
	services := c.GetServiceList(filter)

	for _, srv := range services {
		c.logger.WithFields(mon.Fields{
			"application": srv.Application,
			"eventType":   srv.EventType,
		}).Info("stopping service")
	}

	c.ScaleServices(filter, count)
	c.WaitUntilServiceIsStable(filter)

	return c.GetServices(filter)
}

func (c *AwsServiceClient) Stop(filter *FilterServicesInput) ([]*ecs.Service, error) {
	services := c.GetServiceList(filter)

	for _, srv := range services {
		c.logger.WithFields(mon.Fields{
			"application": srv.Application,
			"eventType":   srv.EventType,
		}).Info("stopping service")
	}

	c.ScaleServices(filter, 0)
	c.WaitUntilServiceIsStable(filter)

	return c.GetServices(filter)
}

func (c *AwsServiceClient) ScaleServices(filter *FilterServicesInput, count int) {
	services, err := c.GetServices(filter)

	if err != nil {
		return
	}

	for _, srv := range services {
		info := c.GetListingFromArn(srv.ServiceArn)
		input := ecs.UpdateServiceInput{
			Cluster:      srv.ClusterArn,
			Service:      srv.ServiceName,
			DesiredCount: aws.Int64(int64(count)),
		}

		_, err := c.client.UpdateService(&input)

		if err != nil {
			c.logger.WithFields(mon.Fields{
				"application": info.Application,
				"eventType":   info.EventType,
			}).Error(err, "could not scale service")

			continue
		}

		c.logger.WithFields(mon.Fields{
			"application": info.Application,
			"eventType":   info.EventType,
		}).Info("scaling service")
	}
}

func (c *AwsServiceClient) ForceNewDeployment(filter *FilterServicesInput) error {
	services, err := c.GetServices(filter)

	if err != nil {
		return err
	}

	for _, srv := range services {
		info := c.GetListingFromArn(srv.ServiceArn)
		input := ecs.UpdateServiceInput{
			Cluster:            srv.ClusterArn,
			Service:            srv.ServiceName,
			ForceNewDeployment: aws.Bool(true),
		}

		_, err := c.client.UpdateService(&input)

		if err != nil {
			c.logger.WithFields(mon.Fields{
				"application": info.Application,
				"eventType":   info.EventType,
			}).Error(err, "could not force deploy the service")

			return err
		}

		c.logger.WithFields(mon.Fields{
			"application": info.Application,
			"eventType":   info.EventType,
		}).Info("force deploying the service")
	}

	return nil
}

func (c *AwsServiceClient) GetServices(filter *FilterServicesInput) ([]*ecs.Service, error) {
	result := make([]*ecs.Service, 0)
	list := c.GetServiceList(filter)

	for i := 0; i < len(list); i += 10 {
		end := i + 10

		if end > len(list) {
			end = len(list)
		}

		arns := make([]*string, 0)
		for _, srv := range list[i:end] {
			arns = append(arns, aws.String(srv.Arn))
		}

		input := ecs.DescribeServicesInput{
			Cluster:  aws.String(c.clusterName),
			Services: arns,
		}

		out, err := c.client.DescribeServices(&input)

		if err != nil {
			c.logger.Error(err, "could not describe services")

			return nil, err
		}

		result = append(result, out.Services...)
	}

	return result, nil
}

func (c *AwsServiceClient) WaitUntilServiceIsStable(filter *FilterServicesInput) {
	c.logger.WithFields(mon.Fields{
		"application": strings.Join(filter.Applications, ","),
		"eventType":   filter.EventType,
	}).Info("waiting for service getting stable")

	list := c.GetServiceList(filter)

	for i := 0; i < len(list); i += 10 {
		end := i + 10

		if end > len(list) {
			end = len(list)
		}

		arns := make([]*string, 0)
		for _, srv := range list[i:end] {
			arns = append(arns, aws.String(srv.Arn))
		}

		input := ecs.DescribeServicesInput{
			Cluster:  aws.String(c.clusterName),
			Services: arns,
		}

		err := c.client.WaitUntilServicesStable(&input)

		if err != nil {
			c.logger.Error(err, "could not wait until services are stable")
			return
		}
	}

	c.logger.WithFields(mon.Fields{
		"application": strings.Join(filter.Applications, ","),
		"eventType":   filter.EventType,
	}).Info("services are stable")
}

func (c *AwsServiceClient) GetServiceList(filter *FilterServicesInput) []ServiceListing {
	input := ecs.ListServicesInput{
		Cluster: aws.String(c.clusterName),
	}

	services := make([]ServiceListing, 0)

	for {
		out, err := c.client.ListServices(&input)

		if err != nil {
			c.logger.Error(err, "could not get the list of services")
			break
		}

		for _, srv := range out.ServiceArns {
			listing := c.GetListingFromArn(srv)

			isApplication := len(filter.Applications) == 0 || funk.ContainsString(filter.Applications, listing.Application)
			isEventType := len(filter.EventType) == 0 || funk.ContainsString(filter.EventType, listing.EventType)

			if isApplication && isEventType {
				services = append(services, listing)
			}
		}

		if out.NextToken == nil {
			break
		}

		input.SetNextToken(*out.NextToken)
	}

	return services
}

func (c *AwsServiceClient) GetListingFromArn(arn *string) ServiceListing {
	r, err := regexp.Compile("([^\\/]*)\\/([^-]*)(-.*)?")

	if err != nil {
		panic(err)
	}

	matches := r.FindStringSubmatch(*arn)

	listing := ServiceListing{
		Arn:         matches[0],
		Name:        matches[2] + matches[3],
		Application: matches[2],
		EventType:   strings.TrimPrefix(matches[3], "-"),
	}

	return listing
}
