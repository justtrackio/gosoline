package cloud_test

import (
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestGetDiscoveryClient(t *testing.T) {
	config := new(cfgMocks.Config)
	config.On("GetString", "aws_serviceDiscovery_endpoint").Return("127.0.0.1")
	config.On("GetString", "aws_serviceDiscovery_namespace").Return("dev")
	logger := logMocks.NewLoggerMockedAll()

	//this also tests *WithInterfaces
	_ = cloud.GetDiscoveryClient(config, logger)

	config.AssertExpectations(t)
}

func TestDiscoveryClient_GetServiceInstances(t *testing.T) {
	configMock := new(cfgMocks.Config)
	configMock.On("GetString", mock.Anything).Return("")

	loggerMock := new(logMocks.Logger)
	clientMock := new(cloudMocks.ServiceDiscoveryAPI)

	clientMock.On("ListServices", mock.AnythingOfType("*servicediscovery.ListServicesInput")).Return(&servicediscovery.ListServicesOutput{
		Services: []*servicediscovery.ServiceSummary{{
			Arn:  aws.String("srvArn1"),
			Name: aws.String("srvName1"),
			Id:   aws.String("srvId1"),
		}},
	}, nil)

	clientMock.On("ListInstances", mock.AnythingOfType("*servicediscovery.ListInstancesInput")).Return(&servicediscovery.ListInstancesOutput{
		Instances: []*servicediscovery.InstanceSummary{{
			Id: aws.String("srvId1"),
			Attributes: map[string]*string{
				"AWS_INSTANCE_IPV4": aws.String("127.0.0.1"),
				"AWS_INSTANCE_PORT": aws.String("8080"),
			},
		}},
	}, nil)

	discoveryClient := cloud.GetDiscoveryClientWithInterfaces(clientMock, configMock, loggerMock)
	instances := discoveryClient.GetServiceInstances("srv")

	assert.Len(t, instances, 1)
	configMock.AssertExpectations(t)
}

func TestDiscoveryClient_GetPipelineInstances(t *testing.T) {
	configMock := new(cfgMocks.Config)
	configMock.On("GetString", mock.Anything).Return("")

	loggerMock := new(logMocks.Logger)
	clientMock := new(cloudMocks.ServiceDiscoveryAPI)

	serviceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"
	serviceName := "aggregator-some-event-type"
	serviceId := "123"
	awsInstanceIpv4 := "127.0.0.1"
	awsInstancePort := "8080"

	clientMock.On("ListServices", mock.AnythingOfType("*servicediscovery.ListServicesInput")).Return(&servicediscovery.ListServicesOutput{
		Services: []*servicediscovery.ServiceSummary{{
			Arn:  aws.String(serviceArn),
			Name: aws.String(serviceName),
			Id:   aws.String(serviceId),
		}},
	}, nil)

	clientMock.On("ListInstances", mock.AnythingOfType("*servicediscovery.ListInstancesInput")).Return(&servicediscovery.ListInstancesOutput{
		Instances: []*servicediscovery.InstanceSummary{{
			Id: aws.String(serviceId),
			Attributes: map[string]*string{
				"AWS_INSTANCE_IPV4": aws.String(awsInstanceIpv4),
				"AWS_INSTANCE_PORT": aws.String(awsInstancePort),
			},
		}},
	}, nil)

	application := "aggregator"

	discoveryClient := cloud.GetDiscoveryClientWithInterfaces(clientMock, configMock, loggerMock)
	instances := discoveryClient.GetPipelineInstances(application)

	assert.Len(t, instances, 2)
	assert.Len(t, instances["some-event-type"], 1)
	assert.Equal(t, awsInstanceIpv4+":"+awsInstancePort, instances["some-event-type"][0])
	configMock.AssertExpectations(t)
	clientMock.AssertExpectations(t)
}
