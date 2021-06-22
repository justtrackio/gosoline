package cloud_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/applike/gosoline/pkg/cloud"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestGetServiceClient(t *testing.T) {
	assert.NotPanics(t, func() {
		loggerMock := logMocks.NewLoggerMockedAll()
		clientMock := new(cloudMocks.ECSAPI)
		appId := &cfg.AppId{}

		serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)
		assert.NotNil(t, serviceClient)
	})
}

func TestGetServiceClientWithDefaultClient(t *testing.T) {
	assert.NotPanics(t, func() {
		config := new(mocks.Config)
		config.On("GetString", "app_project").Return("")
		config.On("GetString", "env").Return("")
		config.On("GetString", "app_family").Return("")
		config.On("GetString", "app_name").Return("")
		loggerMock := logMocks.NewLoggerMockedAll()

		serviceClient := cloud.GetServiceClientWithDefaultClient(config, loggerMock)
		assert.NotNil(t, serviceClient)
	})
}

func TestServiceClient_Stop(t *testing.T) {
	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceArn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{mdl.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Services: []*ecs.Service{{
				ServiceArn:  mdl.String(serviceArn),
				ServiceName: mdl.String("test-service"),
				Tags: []*ecs.Tag{
					{
						Key:   mdl.String("foo"),
						Value: mdl.String("bar"),
					},
				},
			}},
		},
		nil,
	)

	clientMock.On("UpdateService", mock.AnythingOfType("*ecs.UpdateServiceInput")).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceArn:  &serviceArn,
				ServiceName: mdl.String("test-service"),
				Tags: []*ecs.Tag{
					{
						Key:   mdl.String("foo"),
						Value: mdl.String("bar"),
					},
				},
			},
		},
		nil,
	).Once()

	clientMock.On("WaitUntilServicesStable", mock.Anything).Return(nil).Once()

	filter := &cloud.FilterServicesInput{
		Tags: map[string][]string{
			"foo": {"bar"},
		},
	}

	assert.NotPanics(t, func() {
		appId := &cfg.AppId{}

		serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)
		services, err := serviceClient.Stop(filter)

		assert.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, serviceArn, *services[0].ServiceArn)
	})
}

func TestServiceClient_ScaleServices(t *testing.T) {
	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)
	appId := &cfg.AppId{}

	serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)

	serviceArn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"
	count := 3

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{mdl.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn:  mdl.String(serviceArn),
				ServiceName: mdl.String("test-service"),
			}},
		},
		nil,
	).Once()

	clientMock.On("UpdateService", mock.AnythingOfType("*ecs.UpdateServiceInput")).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{},
		},
		nil,
	).Once()

	filter := &cloud.FilterServicesInput{
		Tags: map[string][]string{
			"foo": {"bar"},
		},
	}

	assert.NotPanics(t, func() {
		serviceClient.ScaleServices(filter, count)
	})
}

func TestServiceClient_ForceNewDeployment(t *testing.T) {
	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)
	appId := &cfg.AppId{}

	serviceArn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{mdl.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn:  mdl.String(serviceArn),
				ServiceName: mdl.String("test-service"),
			}},
		},
		nil,
	).Once()

	clientMock.On("UpdateService", mock.AnythingOfType("*ecs.UpdateServiceInput")).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceArn: mdl.String(serviceArn),
			},
		},
		nil,
	).Once()

	filter := &cloud.FilterServicesInput{
		Tags: map[string][]string{
			"foo": {"bar"},
		},
	}

	assert.NotPanics(t, func() {
		serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)
		err := serviceClient.ForceNewDeployment(filter)

		assert.NoError(t, err)
	})
}

func TestServiceClient_GetServices(t *testing.T) {
	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	appId := &cfg.AppId{}

	serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)

	serviceArn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"
	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{
				mdl.String(serviceArn),
			},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn:  mdl.String(serviceArn),
				ServiceName: mdl.String("test-service"),
				Tags:        []*ecs.Tag{},
			}},
		},
		nil,
	).Twice()

	filter := &cloud.FilterServicesInput{
		Tags: map[string][]string{},
	}

	assert.NotPanics(t, func() {
		services, err := serviceClient.GetServices(filter)

		assert.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, serviceArn, *services[0].ServiceArn)
	})
}

func TestServiceClient_WaitUntilServiceIsStable(t *testing.T) {
	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	appId := &cfg.AppId{}

	serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)

	filter := &cloud.FilterServicesInput{
		Tags: map[string][]string{
			"foo": {"bar"},
		},
	}

	myServiceArn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"

	myServicesStrings := make([]*string, 1)
	myServicesStrings[0] = &myServiceArn

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: myServicesStrings,
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Services: []*ecs.Service{{
				ServiceArn:  mdl.String(myServiceArn),
				ServiceName: mdl.String("test-service"),
				Tags: []*ecs.Tag{
					{
						Key:   mdl.String("foo"),
						Value: mdl.String("bar"),
					},
				},
			}},
		},
		nil,
	)

	clientMock.On("WaitUntilServicesStable", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(nil).Once()

	assert.NotPanics(t, func() {
		serviceClient.WaitUntilServiceIsStable(filter)

		clientMock.AssertCalled(t, "WaitUntilServicesStable", mock.Anything)
		loggerMock.AssertNotCalled(t, "Error", mock.AnythingOfType("error"), mock.Anything)
	})
}

func TestServiceClient_GetListingFromArn(t *testing.T) {
	arn := "arn:aws:ecs:eu-central-1:123456789012:service/test-cluster/test-service"

	loggerMock := logMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)
	clientMock.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster: mdl.String("my-test-cluster"),
		Services: []*string{
			mdl.String(arn),
		},
		Include: []*string{mdl.String(ecs.ServiceFieldTags)},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				ServiceArn:  mdl.String(arn),
				ServiceName: mdl.String("test-service"),
				Tags:        []*ecs.Tag{},
			},
		},
	}, nil)

	appId := &cfg.AppId{
		Project:     "my",
		Environment: "test",
		Family:      "cluster",
	}

	assert.NotPanics(t, func() {
		serviceClient := cloud.GetServiceClient(loggerMock, clientMock, appId)
		serviceListing, err := serviceClient.GetListingFromArn(mdl.String(arn))

		assert.Equal(t, arn, serviceListing.Arn)
		assert.Equal(t, "test-service", serviceListing.Name)
		assert.NoError(t, err)
	})
}
