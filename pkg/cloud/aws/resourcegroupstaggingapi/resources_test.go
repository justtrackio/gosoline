package resourcegroupstaggingapi_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	gosoRes "github.com/justtrackio/gosoline/pkg/cloud/aws/resourcegroupstaggingapi"
	gosoResMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/resourcegroupstaggingapi/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestResourcesManager_GetResources(t *testing.T) {
	ctx := context.Background()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	client := gosoResMocks.NewClient(t)
	client.EXPECT().GetResources(ctx, mock.AnythingOfType("*resourcegroupstaggingapi.GetResourcesInput"), mock.AnythingOfType("func(*resourcegroupstaggingapi.Options)")).Return(&resourcegroupstaggingapi.GetResourcesOutput{
		PaginationToken: nil,
		ResourceTagMappingList: []types.ResourceTagMapping{{
			ResourceARN: aws.String("arn:aws:sqs:region:accountId:justtrack-test-gosoline-queue-id"),
			Tags:        nil,
		}},
	}, nil)

	srv := gosoRes.NewServiceWithInterfaces(client, logger)
	r, err := srv.GetResources(ctx, gosoRes.Filter{
		ResourceFilter: nil,
		TagFilter:      nil,
	})

	expected := []string{"arn:aws:sqs:region:accountId:justtrack-test-gosoline-queue-id"}

	assert.NoError(t, err)
	assert.Equal(t, expected, r)
}
