package resources

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
)

//go:generate mockery -name Service
type Service interface {
	GetResources(filter Filter) ([]string, error)
}

type Filter struct {
	ResourceFilter []string
	TagFilter      map[string][]string
}

type resourceManager struct {
	client resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	logger log.Logger
}

func NewService(config cfg.Config, logger log.Logger) Service {
	client := GetClient(config, logger)

	return NewServiceWithInterfaces(client, logger)
}

func NewServiceWithInterfaces(client resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI, logger log.Logger) Service {
	return &resourceManager{
		client: client,
		logger: logger,
	}
}

func (m *resourceManager) GetResources(filter Filter) ([]string, error) {
	input := buildGetResourceInput(filter)
	arns := make([]string, 0)

	err := m.client.GetResourcesPages(input, func(output *resourcegroupstaggingapi.GetResourcesOutput, lastPage bool) bool {
		for _, rtm := range output.ResourceTagMappingList {
			arns = append(arns, *rtm.ResourceARN)
		}

		return !lastPage
	})

	return arns, err
}

func buildGetResourceInput(filter Filter) *resourcegroupstaggingapi.GetResourcesInput {
	var tagFilters []*resourcegroupstaggingapi.TagFilter
	var resourceFilters []*string

	if filter.TagFilter != nil {
		for tag, value := range filter.TagFilter {
			tagFilters = append(tagFilters, &resourcegroupstaggingapi.TagFilter{
				Key:    aws.String(tag),
				Values: aws.StringSlice(value),
			})
		}
	}

	if filter.ResourceFilter != nil {
		resourceFilters = aws.StringSlice(filter.ResourceFilter)
	}

	return &resourcegroupstaggingapi.GetResourcesInput{
		ResourceTypeFilters: resourceFilters,
		TagFilters:          tagFilters,
	}
}
