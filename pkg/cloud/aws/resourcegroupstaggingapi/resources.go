package resourcegroupstaggingapi

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Service
type Service interface {
	GetResources(ctx context.Context, filter Filter) ([]string, error)
}

type Filter struct {
	ResourceFilter []string
	TagFilter      map[string][]string
}

type resourceManager struct {
	client Client
	logger log.Logger
}

func NewService(ctx context.Context, config cfg.Config, logger log.Logger) (Service, error) {
	client, err := ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return NewServiceWithInterfaces(client, logger), nil
}

func NewServiceWithInterfaces(client Client, logger log.Logger) Service {
	return &resourceManager{
		client: client,
		logger: logger,
	}
}

func (m *resourceManager) GetResources(ctx context.Context, filter Filter) ([]string, error) {
	input := buildGetResourceInput(filter)
	arns := make([]string, 0)

	paginator := resourcegroupstaggingapi.NewGetResourcesPaginator(m.client, input, func(options *resourcegroupstaggingapi.GetResourcesPaginatorOptions) {
		options.StopOnDuplicateToken = true
	})

	for paginator.HasMorePages() {
		var err error
		var output *resourcegroupstaggingapi.GetResourcesOutput

		if output, err = paginator.NextPage(ctx); err != nil {
			return nil, fmt.Errorf("can not get next page of resources: %w", err)
		}

		for _, rtm := range output.ResourceTagMappingList {
			arns = append(arns, *rtm.ResourceARN)
		}
	}

	return arns, nil
}

func buildGetResourceInput(filter Filter) *resourcegroupstaggingapi.GetResourcesInput {
	var tagFilters []types.TagFilter
	var resourceFilters []string

	if filter.TagFilter != nil {
		for tag, value := range filter.TagFilter {
			tagFilters = append(tagFilters, types.TagFilter{
				Key:    aws.String(tag),
				Values: value,
			})
		}
	}

	if filter.ResourceFilter != nil {
		resourceFilters = filter.ResourceFilter
	}

	return &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: resourceFilters,
		TagFilters:          tagFilters,
	}
}
