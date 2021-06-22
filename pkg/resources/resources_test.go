package resources_test

import (
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/resources"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestResourcesManager_GetResources(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()

	client := new(cloudMocks.ResourceGroupsTaggingAPIAPI)
	client.On("GetResourcesPages",
		mock.AnythingOfType("*resourcegroupstaggingapi.GetResourcesInput"),
		mock.AnythingOfType("func(*resourcegroupstaggingapi.GetResourcesOutput, bool) bool")).Run(func(args mock.Arguments) {
		callback := args[1].(func(*resourcegroupstaggingapi.GetResourcesOutput, bool) bool)
		callback(&resourcegroupstaggingapi.GetResourcesOutput{
			PaginationToken: nil,
			ResourceTagMappingList: []*resourcegroupstaggingapi.ResourceTagMapping{{
				ResourceARN: aws.String("arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id"),
				Tags:        nil,
			}},
		}, true)
	}).Return(nil)

	srv := resources.NewServiceWithInterfaces(client, logger)
	r, err := srv.GetResources(resources.Filter{
		ResourceFilter: nil,
		TagFilter:      nil,
	})

	expected := []string{"arn:aws:sqs:region:accountId:applike-test-gosoline-queue-id"}

	assert.NoError(t, err)
	assert.Equal(t, expected, r)

	client.AssertExpectations(t)
}
