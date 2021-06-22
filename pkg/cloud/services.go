package cloud

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/thoas/go-funk"
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
	GetListingFromArn(arn *string) (*ServiceListing, error)
	GetListingFromService(svc *ecs.Service) *ServiceListing
}

type AwsServiceClient struct {
	logger      log.Logger
	client      ecsiface.ECSAPI
	clusterName string
}

type ServiceListing struct {
	Arn  string
	Name string
	Tags map[string]string
}

func GetServiceClient(logger log.Logger, client ecsiface.ECSAPI, appId *cfg.AppId) ServiceClient {
	clusterName := fmt.Sprintf("%s-%s-%s", appId.Project, appId.Environment, appId.Family)
	logger = logger.WithFields(log.Fields{
		"clusterName": clusterName,
	})

	return &AwsServiceClient{
		logger:      logger,
		client:      client,
		clusterName: clusterName,
	}
}

func GetServiceClientWithDefaultClient(config cfg.Config, logger log.Logger) ServiceClient {
	client := GetEcsClient(logger)

	appId := &cfg.AppId{}
	appId.PadFromConfig(config)

	return GetServiceClient(logger, client, appId)
}

func (c *AwsServiceClient) SetClient(client ecsiface.ECSAPI) {
	c.client = client
}

type FilterServicesInput struct {
	Tags map[string][]string
}

type ScaleServicesInput struct {
	Filter FilterServicesInput `binding:"required"`
	Count  int                 `binding:"required"`
}

func (c *AwsServiceClient) Start(filter *FilterServicesInput, count int) ([]*ecs.Service, error) {
	c.ScaleServices(filter, count)
	c.WaitUntilServiceIsStable(filter)

	return c.GetServices(filter)
}

func (c *AwsServiceClient) Stop(filter *FilterServicesInput) ([]*ecs.Service, error) {
	c.ScaleServices(filter, 0)
	c.WaitUntilServiceIsStable(filter)

	return c.GetServices(filter)
}

func (c *AwsServiceClient) ScaleServices(filter *FilterServicesInput, count int) {
	logger := c.logger.WithFields(c.getLoggerFields(filter))
	services, err := c.GetServices(filter)

	if err != nil {
		return
	}

	for _, srv := range services {
		input := ecs.UpdateServiceInput{
			Cluster:      srv.ClusterArn,
			Service:      srv.ServiceName,
			DesiredCount: aws.Int64(int64(count)),
		}

		_, err := c.client.UpdateService(&input)

		if err != nil {
			logger.Error("could not scale service: %w", err)

			continue
		}

		logger.WithFields(log.Fields{
			"desired_count": count,
		}).Info("scaling service")
	}
}

func (c *AwsServiceClient) ForceNewDeployment(filter *FilterServicesInput) error {
	logger := c.logger.WithFields(c.getLoggerFields(filter))
	services, err := c.GetServices(filter)

	if err != nil {
		return err
	}

	for _, srv := range services {
		input := ecs.UpdateServiceInput{
			Cluster:            srv.ClusterArn,
			Service:            srv.ServiceName,
			ForceNewDeployment: aws.Bool(true),
		}

		_, err := c.client.UpdateService(&input)

		if err != nil {
			logger.Error("could not force deploy the service: %w", err)

			return err
		}

		logger.Info("force deploying the service")
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
			Include:  []*string{mdl.String(ecs.ServiceFieldTags)},
			Services: arns,
		}

		out, err := c.client.DescribeServices(&input)

		if err != nil {
			c.logger.Error("could not describe services: %w", err)

			return nil, err
		}

		result = append(result, out.Services...)
	}

	return result, nil
}

func (c *AwsServiceClient) WaitUntilServiceIsStable(filter *FilterServicesInput) {
	logger := c.logger.WithFields(c.getLoggerFields(filter))
	logger.Info("waiting for service getting stable")

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
			logger.Error("could not wait until services are stable: %w", err)
			return
		}
	}

	logger.Info("services are stable")
}

func (c *AwsServiceClient) GetServiceList(filter *FilterServicesInput) []ServiceListing {
	input := ecs.ListServicesInput{
		Cluster: aws.String(c.clusterName),
	}

	services := make([]ServiceListing, 0)

	for {
		out, err := c.client.ListServices(&input)

		if err != nil {
			c.logger.Error("could not get the list of services: %w", err)

			break
		}

		for _, srv := range out.ServiceArns {
			listing, err := c.GetListingFromArn(srv)

			if err != nil {
				c.logger.Error("failed to get listing for arn: %w", err)

				return nil
			}

			hasAllTags := c.hasTags(listing, filter.Tags)

			if !hasAllTags {
				continue
			}

			services = append(services, *listing)
		}

		if out.NextToken == nil {
			break
		}

		input.SetNextToken(*out.NextToken)
	}

	return services
}

func (c *AwsServiceClient) GetListingFromArn(arn *string) (*ServiceListing, error) {
	svc, err := c.client.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: &c.clusterName,
		Include: []*string{
			mdl.String(ecs.ServiceFieldTags),
		},
		Services: []*string{
			arn,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(svc.Services) > 1 {
		return nil, errors.New("there is more than one service with the specified arn")
	}

	if len(svc.Services) == 0 {
		return nil, errors.New("there is no service with the specified arn")
	}

	return c.GetListingFromService(svc.Services[0]), nil
}

func (c *AwsServiceClient) GetListingFromService(svc *ecs.Service) *ServiceListing {
	tags := map[string]string{}
	funk.ForEach(svc.Tags, func(tag *ecs.Tag) {
		tags[*tag.Key] = *tag.Value
	})

	listing := &ServiceListing{
		Arn:  *svc.ServiceArn,
		Name: *svc.ServiceName,
		Tags: tags,
	}

	return listing
}

func (c *AwsServiceClient) getLoggerFields(filter *FilterServicesInput) log.Fields {
	fields := log.Fields{}

	for key, value := range filter.Tags {
		fields[key] = value
	}

	return fields
}

func (c *AwsServiceClient) hasTags(listing *ServiceListing, tags map[string][]string) bool {
	if listing == nil {
		return false
	}

	for key, value := range tags {
		v, ok := listing.Tags[key]

		if !ok {
			return false
		}

		foundValue := false

		for _, subValue := range value {
			if subValue == v {
				foundValue = true

				break
			}
		}

		if !foundValue {
			return false
		}
	}

	return true
}
