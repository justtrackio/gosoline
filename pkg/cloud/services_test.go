package cloud_test

import (
	"github.com/applike/gosoline/pkg/cloud"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestGetServiceClient(t *testing.T) {
	assert.NotPanics(t, func() {
		loggerMock := monMocks.NewLoggerMockedAll()
		clientMock := new(cloudMocks.ECSAPI)

		serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")
		assert.NotNil(t, serviceClient)
	})
}

func TestGetServiceClientWithDefaultClient(t *testing.T) {
	assert.NotPanics(t, func() {
		loggerMock := monMocks.NewLoggerMockedAll()

		serviceClient := cloud.GetServiceClientWithDefaultClient(loggerMock, "sandbox")
		assert.NotNil(t, serviceClient)
	})
}

func TestServiceClient_Stop(t *testing.T) {
	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{aws.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Services: []*ecs.Service{{
				ServiceArn: aws.String(serviceArn),
			}},
		},
		nil,
	).Twice()

	clientMock.On("UpdateService", mock.AnythingOfType("*ecs.UpdateServiceInput")).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceArn: &serviceArn,
			},
		},
		nil,
	).Once()

	clientMock.On("WaitUntilServicesStable", mock.Anything).Return(nil).Once()

	filter := &cloud.FilterServicesInput{
		Applications: []string{"mcoins"},
		EventType:    []string{"test-ec2"},
	}

	assert.NotPanics(t, func() {
		serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")
		services, err := serviceClient.Stop(filter)

		assert.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, serviceArn, *services[0].ServiceArn)
	})
}

func TestServiceClient_ScaleServices(t *testing.T) {
	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")

	serviceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"
	count := 3

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{aws.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn: aws.String(serviceArn),
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
		Applications: []string{"mcoins"},
		EventType:    []string{"test-ec2"},
	}

	assert.NotPanics(t, func() {
		serviceClient.ScaleServices(filter, count)
	})
}

func TestServiceClient_ForceNewDeployment(t *testing.T) {
	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{aws.String(serviceArn)},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn: aws.String(serviceArn),
			}},
		},
		nil,
	).Once()

	clientMock.On("UpdateService", mock.AnythingOfType("*ecs.UpdateServiceInput")).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceArn: aws.String(serviceArn),
			},
		},
		nil,
	).Once()

	filter := &cloud.FilterServicesInput{
		Applications: []string{"mcoins"},
		EventType:    []string{"test-ec2"},
	}

	assert.NotPanics(t, func() {
		serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")
		err := serviceClient.ForceNewDeployment(filter)

		assert.NoError(t, err)
	})
}

func TestServiceClient_GetServices(t *testing.T) {
	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")

	serviceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"
	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: []*string{
				aws.String(serviceArn),
			},
		},
		nil,
	)

	clientMock.On("DescribeServices", mock.AnythingOfType("*ecs.DescribeServicesInput")).Return(
		&ecs.DescribeServicesOutput{
			Failures: nil,
			Services: []*ecs.Service{{
				ServiceArn: aws.String(serviceArn),
			}},
		},
		nil,
	).Once()

	filter := &cloud.FilterServicesInput{
		Applications: []string{"mcoins"},
		EventType:    []string{"test-ec2"},
	}

	assert.NotPanics(t, func() {
		services, err := serviceClient.GetServices(filter)

		assert.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, serviceArn, *services[0].ServiceArn)
	})
}

func TestServiceClient_WaitUntilServiceIsStable(t *testing.T) {
	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")

	filter := &cloud.FilterServicesInput{
		Applications: []string{"mcoins"},
		EventType:    []string{"test-ec2"},
	}

	myServiceArn := "arn:aws:iam::12345678:role/mcoins-test-ec2"

	myServicesStrings := make([]*string, 1)
	myServicesStrings[0] = &myServiceArn

	clientMock.On("ListServices", mock.AnythingOfType("*ecs.ListServicesInput")).Return(
		&ecs.ListServicesOutput{
			ServiceArns: myServicesStrings,
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
	arn := "arn:aws:iam::12345678:role/mcoins-test-ec2"

	loggerMock := monMocks.NewLoggerMockedAll()
	clientMock := new(cloudMocks.ECSAPI)

	assert.NotPanics(t, func() {
		serviceClient := cloud.GetServiceClient(clientMock, loggerMock, "sandbox")
		serviceListing := serviceClient.GetListingFromArn(aws.String(arn))

		assert.Equal(t, arn, serviceListing.Arn)
		assert.Equal(t, "mcoins-test-ec2", serviceListing.Name)
		assert.Equal(t, "mcoins", serviceListing.Application)
		assert.Equal(t, "test-ec2", serviceListing.EventType)
	})
}
